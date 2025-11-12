#!groovy
library identifier: 'dtp-jenkins-library@main',
    retriever: modernSCM([
        $class: 'GitSCMSource',
        remote: 'https://bitbucket.lab.dynatrace.org/scm/pfi/dtp-jenkins-library.git',
        credentialsId: 'bitbucket-buildmaster'])

def config = [
        skipDefaultCheckout: true,
        semanticVersioning: false,
        docker: [enabled: true],
        helm: [enabled: true, chartDir: "charts/vault-config-operator"],
        ociRegistries: [harbor: false, ecr: true, dtp_harbor: [enabled: false]]
]

dtpPipeline.extendWith(config, customValidationStages(), [:], [:])

def customValidationStages() {
    // Run helm chart generation as a validation stage (runs before Build stage)
    return ["Generate Helm Chart": {
        stage("Generate Helm Chart") {
            container('executor') {
                sh """
                    # Read version from VERSION.txt file
                    VERSION=0.0.0
                    
                    # Install Go if not available
                    cd /tmp
                    if ! command -v go &> /dev/null; then
                        wget -q https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
                        tar -xzf go1.22.0.linux-amd64.tar.gz
                    fi
                    
                    cd \${WORKSPACE}
                    export PATH=/tmp/go/bin:\$PATH
                    export GOPATH=/tmp/go-workspace
                    export IMG=\${CONTAINER_IMAGE_NAME}:\${VERSION}
                    
                    make helmchart OPERATOR_NAME=vault-config-operator VERSION=\${VERSION}
                """
            }
        }
    }]
}
