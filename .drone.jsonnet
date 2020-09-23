local PipelineTesting = {
  kind: 'pipeline',
  name: 'testing',
  type: 'docker',
  steps: [
    {
      name: 'test',
      image: 'golang',
      volumes: [
        {
          name: 'gopath',
          path: '/go',
        },
      ],
      pull: 'always',
      environment: {
        GO111MODULE: 'on',
      },
      commands: [
        'go test',
      ],
    },
  ],
  volumes: [
    {
      name: 'gopath',
      host: {
        path: '/mnt/go',
      },
    },
  ],
  trigger: {
    branch: ['master'],
  },
};
local PipelineBuild(os='linux', arch='amd64') = {
  kind: 'pipeline',
  name: os + '_' + arch,
  type: 'docker',
  strigger: {
    branch: ['master'],
  },
  depends_on: [
    'testing',
  ],
  steps: [
    {
      name: 'build-push',
      image: 'golang',
      volumes: [
        {
          name: 'gopath',
          path: '/go',
        },
      ],
      pull: 'always',
      environment: {
        CGO_ENABLED: '0',
        GO111MODULE: 'on',
      },
      commands: [
        'GOOS=' + os + ' ' + 'GOARCH=' + arch + ' ' + 'CGO_ENABLED=0 ' +
        'go build -v -ldflags "-s -w -X main.version=git-${DRONE_COMMIT_SHA:0:8}" -a -o build/${DRONE_REPO_NAME}_' + os + '_' + arch,
      ],
      when: {
        event: {
          exclude: ['tag'],
        },
      },
    },
    {
      name: 'build',
      image: 'golang',
      pull: 'always',
      environment: {
        GO111MODULE: 'on',
      },
      commands: [
        'GOOS=' + os + ' ' + 'GOARCH=' + arch + ' ' + 'CGO_ENABLED=0 ' +
        'go build -v -ldflags "-s -w -X main.version=${DRONE_TAG##v}" -a -o release/${DRONE_REPO_NAME}_' + os + '_' + arch,
      ],
      when: {
        event: ['tag'],
      },
    },
    {
      name: 'release',
      image: 'plugins/github-release',
      settings: {
        api_key: { from_secret: 'GITHUB_API_KEY' },
        files: 'release/*',
      },
      when: {
        event: 'tag',
      },
    },
  ],
  volumes: [
    {
      name: 'gopath',
      host: {
        path: '/mnt/go',
      },
    },
  ],
  trigger: {
    branch: ['master'],
  },
};
[
  PipelineTesting,
  PipelineBuild('linux', 'amd64'),
  PipelineBuild('linux', 'arm64'),
  PipelineBuild('linux', 'arm'),
]
