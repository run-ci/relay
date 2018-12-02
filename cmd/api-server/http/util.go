package http

import (
	"encoding/json"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

func writeErrResp(rw http.ResponseWriter, err error, status int) {
	rw.WriteHeader(status)

	buf, err := json.Marshal(map[string]string{
		"error": err.Error(),
	})
	if err != nil {
		return
	}

	rw.Write(buf)
	return
}

func sendWithBackoff(logger *logrus.Entry, ch chan<- []byte, msg []byte) {
	jittersrc := rand.NewSource(time.Now().Unix())
	jitter := rand.New(jittersrc)

	for i := 0; i < 5; i++ {
		select {
		case ch <- msg:
			logger.Debug("message sent")
			return
		default:
			base := math.Pow(float64(2), float64(i))
			backoff := time.Duration(jitter.Intn(int(base))) * time.Second

			logger.Warnf("unable to send poller create message, sleeping for %v", backoff)
			time.Sleep(backoff)
		}
	}
}
