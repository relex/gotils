pipeline {
    agent {
        docker { image 'golang:1.16' }
    }
    stages {
        stage('Test') {
            steps {
                sh 'go test -v ./...'
            }
        }
    }
}