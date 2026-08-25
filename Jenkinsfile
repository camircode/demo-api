// Builds the image, pushes it to GHCR by digest, and commits that digest to the
// GitOps repository.
//
// This pipeline never touches the cluster. Argo CD does the deploying, and the
// only thing Jenkins changes is a line in git — which is why the git log of
// camircode/gitops is the real deployment history.

pipeline {
    agent { label 'docker' }

    options {
        timestamps()
        timeout(time: 20, unit: 'MINUTES')
        disableConcurrentBuilds()
        buildDiscarder(logRotator(numToKeepStr: '30'))
    }

    environment {
        REGISTRY = 'ghcr.io'
        IMAGE    = 'ghcr.io/camircode/demo-api'
        GITOPS   = 'git@github.com:camircode/gitops.git'
        MANIFEST = 'manifests/demo-api/deployment.yaml'
    }

    stages {
        stage('Test') {
            steps {
                // In a container rather than on the agent, so the agent does not
                // accumulate a toolchain per language it ever built.
                sh '''
                    docker run --rm -v "$PWD":/src -w /src golang:1.24-alpine \
                      sh -c 'go vet ./... && go test -race -count=1 ./...'
                '''
            }
        }

        stage('Build and push') {
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'ghcr',
                    usernameVariable: 'GHCR_USER',
                    passwordVariable: 'GHCR_PASS',
                )]) {
                    sh '''
                        set -eu
                        echo "$GHCR_PASS" | docker login "$REGISTRY" -u "$GHCR_USER" --password-stdin

                        docker buildx create --use --name builder 2>/dev/null || docker buildx use builder

                        # The tag exists so a human can find the build; the digest
                        # is what gets deployed. --metadata-file is how the digest
                        # comes back without a second registry round trip.
                        docker buildx build \
                          --push \
                          --provenance=false \
                          --build-arg VERSION="${GIT_COMMIT:0:7}" \
                          --build-arg COMMIT="${GIT_COMMIT}" \
                          --tag "$IMAGE:${GIT_COMMIT:0:7}" \
                          --metadata-file metadata.json \
                          .
                    '''
                }
                script {
                    def meta = readJSON file: 'metadata.json'
                    env.IMAGE_DIGEST = meta['containerimage.digest']
                    echo "Pushed ${env.IMAGE}@${env.IMAGE_DIGEST}"
                }
            }
        }

        stage('Update the desired state') {
            steps {
                sshagent(credentials: ['gitops-write']) {
                    sh '''
                        set -eu
                        rm -rf gitops
                        GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=accept-new" \
                          git clone --depth 1 "$GITOPS" gitops

                        cd gitops
                        git config user.email "jenkins@camir.tech"
                        git config user.name  "jenkins"

                        # Replace whatever image line is there with this digest.
                        # Matching on the repository rather than on the old value
                        # means a hand-edited manifest is corrected rather than
                        # skipped.
                        sed -i -E "s#image: .*demo-api@sha256:[a-f0-9]+#image: ${IMAGE}@${IMAGE_DIGEST}#" "$MANIFEST"
                        sed -i -E "s#image: traefik/whoami@sha256:[a-f0-9]+#image: ${IMAGE}@${IMAGE_DIGEST}#" "$MANIFEST"

                        if git diff --quiet; then
                          echo "Already at ${IMAGE_DIGEST}; nothing to commit."
                          exit 0
                        fi

                        git add "$MANIFEST"
                        git commit -m "deploy(demo-api): ${IMAGE_DIGEST}

Built from camircode/demo-api@${GIT_COMMIT} by Jenkins build ${BUILD_NUMBER}."
                        GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=accept-new" git push origin main
                    '''
                }
            }
        }
    }

    post {
        always {
            sh 'docker logout ghcr.io || true'
            cleanWs()
        }
    }
}
