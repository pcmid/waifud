#!/usr/bin/env python
import subprocess

import sys


def run_cmd(cmd):
    p = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout = p.communicate()[0].decode('utf-8').strip()
    return stdout


# Get last tag.
def get_version():
    return run_cmd('git describe --abbrev=0 --tags')


def build_linux():
    goos = "linux"
    if sys.argv[1] == "build":
        for arch in ["amd64", "arm64", "arm"]:
            cmd = 'GOOS={goos} GOARCH={arch} go build --ldflags "-s -w -X main.version={version}" -o waifud_{goos}_{arch}'.format(
                arch=arch,
                version=get_version(),
                goos=goos
            )
            print(cmd)
            run_cmd(cmd)

    elif sys.argv[1] == "package":
        for arch in ["amd64", "arm64", "arm"]:
            cmd = 'tar -zcvf waifud_{goos}_{arch}.tar.gz waifud_{goos}_{arch} config.toml waifud.service LICENSE'.format(
                arch=arch,
                goos=goos,
            )
            print(cmd)
            run_cmd(cmd)


def build_windows():
    goos = "windows"
    arch = "amd64"

    if sys.argv[1] == "build":
        cmd = 'GOOS={goos} OGARCH={arch} go build --ldflags "-s -w -X main.version={version}" -o waifud_{goos}_{arch}.exe'.format(
            goos=goos,
            arch=arch,
            version=get_version(),
        )
        print(cmd)
        run_cmd(cmd)

    elif sys.argv[1] == "package":
        for arch in ["amd64", "arm64", "arm"]:
            cmd = 'zip waifud_{goos}_{arch}.zip waifud_{goos}_{arch}.exe config.toml LICENSE'.format(
                arch=arch,
                goos=goos,
            )
            print(cmd)
            run_cmd(cmd)


def main():
    build_linux()
    build_windows()


if __name__ == '__main__':
    main()
