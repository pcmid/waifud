local PipelineTesting = {
  kind: 'pipeline',
  name: 'testing',
  platform: {
    os: 'linux',
    arch: 'amd64',
  },
  steps: [
    {
      name: 'test',
      image: 'golang',
      pull: 'always',
      environment: {
        GO111MODULE: 'on',
      },
      commands: [
        'go test',
      ],
    },
  ],
  trigger: {
    branch: ['master'],
  },
};
local PipelineBuild(os='linux', arch='amd64') = {
  kind: 'pipeline',
  name: os + '_' + arch,
  strigger: {
    branch: ['master'],
  },
  steps: [
    {
      name: 'build-push',
      image: 'golang',
      pull: 'always',
      environment: {
        CGO_ENABLED: '0',
        GO111MODULE: 'on',
      },
      commands: [
        'GOOS=' + os + ' ' + 'GOARCH=' + arch + ' ' + 'CGO_ENABLED=0 ' +
        'go build -v -ldflags "-s -w -X main.version=${DRONE_COMMIT_SHA:0:8}" -a -o build/${DRONE_REPO_NAME}_' + os + '_' + arch,
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
  depends_on: [
    'testing',
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
