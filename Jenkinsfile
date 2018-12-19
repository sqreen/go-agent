@Library('sqreen-pipeline-library')
import io.sqreen.pipeline.kubernetes.*;
import io.sqreen.pipeline.scm.GitHubSCM;

def gitHub = new GitHubSCM();

String label = templates.generateSlaveName();

public void dockerTemplate(String label, String cloud = 'kubernetes-cluster', Closure body) {
    podTemplate(label: label, cloud: cloud, containers: [
        containerTemplate(name: 'docker', image: 'docker', command: 'cat', ttyEnabled: true)
    ], volumes: [
        hostPathVolume(hostPath: '/var/run/docker.sock', mountPath: '/var/run/docker.sock')
    ]) {
        body();
    }
}

dockerTemplate(label) {
    node(label) {
        stage('Checkout') {
          gitHub.checkoutWithSubModules()
        }
        container('docker') {
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
