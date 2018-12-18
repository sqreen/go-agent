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
                def devImage = docker.build("sqreen/go-agent-dev", "--build-arg uid=${env.UID} -f ./tools/docker/dev/Dockerfile .")
                devImage.inside {
                    sh 'make test'
                }
            }
        }
    }
}
