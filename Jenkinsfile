def name = "yet-another-cloudwatch-exporter"
def app
def version

node {
  stage 'Checkout'
  checkout scm

  stage 'Build image & test'
  app = docker.build "quay.io/invisionag/${name}","--pull ."
  app.push("${env.BRANCH_NAME}-${env.BUILD_NUMBER}")

  if (env.BRANCH_NAME == 'master') {
    stage 'Build Binaries'
    sh 'docker-compose build --pull'
    sh 'docker-compose run app'

    stage 'Dockerhub release'

    timeout(time: 10, unit: 'MINUTES') {
      input 'Release to github?'
    }

    app.inside {
      version = sh (
          returnStdout: true,
          script: "/usr/local/bin/yace -v"
          ).trim()
    }

    app.push(version)

    stage 'Git release'
    sh 'echo "Creating a new release in github"'
    sh "github-release-wrapper release --user ivx --repo ${name} --tag ${version} --name ${version}"

    sh 'echo "Uploading the artifacts into github"'
    sh "github-release-wrapper upload --user ivx --repo ${name} --tag ${version} --name yace-linux-amd64-${version} --file yace-linux-amd64"
    sh "github-release-wrapper upload --user ivx --repo ${name} --tag ${version} --name yace-darwin-amd64-${version} --file yace-darwin-amd64"
  }
}
