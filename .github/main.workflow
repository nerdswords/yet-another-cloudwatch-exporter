workflow "Build, test and publish" {
  on = "push"
  resolves = [
    "Publish docker image",
    "Build && release binaries"
  ]
}

action "Build docker image" {
  uses = "actions/docker/cli@master"
  args = "build -t yace --build-arg VERSION=${GITHUB_REF#refs/tags/} ."
}

action "Release if tagged" {
  needs = ["Build docker image"]
  uses = "actions/bin/filter@master"
  args = "tag v*"
}

action "Release if master branch" {
  needs = ["Release if tagged"]
  uses = "actions/bin/filter@master"
  args = "branch master"
}

action "Build && release binaries" {
  needs = ["Release if master branch"]
  secrets = ["GITHUB_TOKEN"]
  uses = "docker://goreleaser/goreleaser:v0.104"
  args = ["release"]
}

action "Log into docker" {
  needs = ["Release if master branch"]
  uses = "actions/docker/login@master"
  secrets = ["DOCKER_USERNAME", "DOCKER_PASSWORD"]
  env = {
    DOCKER_REGISTRY_URL  = "quay.io"
  }
}

action "Tag docker image" {
  needs = ["Log into docker"]
  uses = "actions/docker/tag@master"
  args = "--no-latest --no-sha yace quay.io/invisionag/yet-another-cloudwatch-exporter"
}

action "Publish docker image" {
  needs = ["Tag docker image"]
  uses = "actions/docker/cli@master"
  args = ["push quay.io/invisionag/yet-another-cloudwatch-exporter"]
}
