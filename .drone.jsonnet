local PipelineTesting = {
  kind: 'pipeline',
  name: 'testing',
  type: 'kubernetes',
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
        GOPROXY: {
          from_secret: 'goproxy',
        },
      },
      commands: [
        'go mod download',
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
    events: ['push'],
  },
};
local PipelineBuild(os='linux', arch='amd64') = {
  kind: 'pipeline',
  name: os + '_' + arch,
  type: 'kubernetes',
  environment: {
    GOPROXY: {
      from_secret: 'goproxy',
    },
  },
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
        GOPROXY: {
          from_secret: 'goproxy',
        },
      },
      commands: [
        'go mod download',
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
        GOPROXY: {
          from_secret: 'goproxy',
        },
      },
      commands: [
        'go mod download',
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
    events: ['pull_request'],
  },
};
[
  PipelineTesting,
  PipelineBuild('linux', 'amd64'),
  PipelineBuild('linux', 'arm64'),
  PipelineBuild('linux', 'arm'),
]
