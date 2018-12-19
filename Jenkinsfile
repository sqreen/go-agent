@Library('sqreen-pipeline-library')
import io.sqreen.pipeline.kubernetes.PodTemplate;
import io.sqreen.pipeline.scm.GitHubSCM;

def templates = new Pod();
def gitHub = new GitHubSCM();

String label = templates.generateSlaveName();

templates.dockerTemplate(label) {
    node(label) {
        container('docker') {
            stage('Checkout') {
                gitHub.checkoutWithSubModules()
            }
            stage('Build') {
                sh 'pwd'
                sh 'ls -a'
                sh 'docker info'
                def devImage = docker.build("sqreen/go-agent-dev", "-f ./tools/docker/dev/Dockerfile .")
                devImage.inside("-v $PWD:$PWD -w $PWD --name go-agent-dev") {
                    sh 'pwd'
                    sh 'ls -a'
                    sh 'make test'
                }
            }
        }
    }
}
