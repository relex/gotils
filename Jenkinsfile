pipeline {
    agent {
        docker {
            image 'golang:1.16'
            args '-u root:root'
        }
    }
    stages {
        stage('Test') {
            steps {
                sh 'go test -v ./...'
            }
        }
    }
}