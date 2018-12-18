@Library('sqreen-pipeline-library')
import io.sqreen.pipeline.kubernetes.*;
import io.sqreen.pipeline.scm.GitHubSCM;

def templates = new PodTemplate();
def gitHub = new GitHubSCM();

String dockerLabel = templates.generateSlaveName();

templates.dockerTemplate(dockerLabel) {
    node(dockerLabel) {
        gitHub.checkoutWithSubModules()
        container('docker') {
            sh "make test"
        }
    }
}
