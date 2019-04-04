@Library('sqreen-pipeline-library')
import io.sqreen.pipeline.kubernetes.PodTemplate;
import io.sqreen.pipeline.scm.GitHubSCM;
import io.sqreen.pipeline.tools.Codecov;

def templates = new PodTemplate();
def gitHub = new GitHubSCM();
def codecov = new Codecov();

String label = 'docker';

templates.dockerTemplate(label) {
    node(label) {
        container('docker') {
            stage('Checkout') {
                gitHub.checkoutWithSubModules()
            }

            sh 'docker info'
            def devImage = docker.build("sqreen/go-agent-dev", "-f ./tools/docker/dev/Dockerfile .")
                devImage.inside("--name go-agent-dev -e GO111MODULE=on -e GOPATH=$WORKSPACE/.cache/go -e GOCACHE=$WORKSPACE/.cache") {
                stage('Vendoring') {
                    sh 'make vendor'
                }

                stage('Tests') {
                    parallel([
                        'Regular': {
                            sh 'go env'
                            sh 'make test'
                        },
                        'With coverage': {
                            sh 'make test-coverage'
                            codecov.analyze('codecov-agent-go-token')
                        },
                        'With race detection': {
                            sh 'make test-race'
                        },
                        'Benchmarks': {
                            sh 'make benchmark'
                        }
                    ])
                }
            }
        }
    }
}
