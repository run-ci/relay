INSERT INTO projects (name)
VALUES
    ('Relay');

INSERT INTO git_remotes (url, branch, project_id)
SELECT 'https://github.com/run-ci/relay.git', 'master', projects.id
FROM projects WHERE projects.name = 'Relay';
