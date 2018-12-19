@Library('sqreen-pipeline-library')
import io.sqreen.pipeline.kubernetes.*;
import io.sqreen.pipeline.scm.GitHubSCM;

def templates = new PodTemplate();
def gitHub = new GitHubSCM();

String label = templates.generateSlaveName();

templates.dockerTemplate(label) {
    node(label) {
        stage('Checkout') {
          gitHub.checkoutWithSubModules()
        }
        container('docker') {
            stage('Build') {
                def devImage = docker.build("sqreen/go-agent-dev", "-f ./tools/docker/dev/Dockerfile .")
                devImage.inside("-v $PWD:$PWD -w $PWD --name go-agent-dev") {
                    sh 'ls -a'
                    sh 'pwd'
                    sh 'make test'
                }
            }
        }
    }
}
