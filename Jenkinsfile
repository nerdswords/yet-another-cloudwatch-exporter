def name = "yet-another-cloudwatch-exporter"
def app

stage 'Checkout'
node {
  checkout scm
}

stage 'Build Image'
node {
  gitCommit = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
  app = docker.build "quay.io/invisionag/${name}","--pull ."
  app.push("${env.BRANCH_NAME}-${env.BUILD_NUMBER}")
}

if (env.BRANCH_NAME == 'master') {
  stage 'Test'
  node {
    sh 'echo "todo"'
  }

  stage 'Build'
    node {
      sh 'docker-compose build --pull'
      sh 'docker-compose run app'
  }

  timeout(time: 1, unit: 'HOURS') {
    input 'Release to github?'
    stage 'Dockerhub release'
    node {
      version = readFile('version.txt').trim()
      app.push(version)
    }

    stage 'Git release'
     node {
      version = readFile('version.txt').trim()

      sh 'echo "Creating a new release in github"'
      sh "github-release-wrapper release --user ivx --repo ${name} --tag ${version} --name ${version}"

      sh 'echo "Uploading the artifacts into github"'
      sh "github-release-wrapper upload --user ivx --repo ${name} --tag ${version} --name yace-linux-amd64-${version} --file yace-linux-amd64"
      sh "github-release-wrapper upload --user ivx --repo ${name} --tag ${version} --name yace-darwin-amd64-${version} --file yace-darwin-amd64"
    }
  }
}
