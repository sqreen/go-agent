@Library('sqreen-pipeline-library')
import io.sqreen.pipeline.kubernetes.PodTemplate;
import io.sqreen.pipeline.scm.GitHubSCM;

def templates = new PodTemplate();
def gitHub = new GitHubSCM();

String label = templates.generateSlaveName();

templates.dockerTemplate(label) {
    node(label) {
        container('docker') {
            stage('Checkout') {
                gitHub.checkoutWithSubModules()
            }

            sh 'docker info'
            def devImage = docker.build("sqreen/go-agent-dev", "-f ./tools/docker/dev/Dockerfile .")
                devImage.inside("--name go-agent-dev -e GOPATH=$WORKSPACE -e GOCACHE=$WORKSPACE/.cache") {
                stage('Test') {
                    sh 'go env'
                    sh 'make test'
                    parallel([
                              'Coverage': {
                                  sh 'make test-with-coverage'
                              }
                            ])
                    }
                }
            }
        }
    }
}
