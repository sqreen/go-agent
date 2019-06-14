@Library('sqreen-pipeline-library')
import io.sqreen.pipeline.tools.*

def DOCKER_IMAGE = 'golang:1'
def GIT_REPO = 'github.com/sqreen/go-agent'
def GO_PACKAGE_PATH_WITHOUT_GOMODULES = '/go/src/${GIT_REPO}'

BuildJob buildJob = new BuildJob(this)
Gradle gradle = new Gradle(this)
Git git = new Git(this)

def projectVersion
def isReleaseBuild

stage('Init') {
    buildJob.setDefaultProperties()
}

node('docker_build') {

    stage('Checkout') {
        // checkout with submodules
        git.checkout()
    }

    stage('Test') {

        docker.image(DOCKER_IMAGE).inside {
            sh 'make test test-race benchmark'
        }

    }

}
