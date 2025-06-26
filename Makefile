ECR_REPO=REDACTED_AWS_ACCOUNT_DEMO.dkr.ecr.ap-northeast-1.amazonaws.com/quantumcat
PLATFORM=linux/amd64

.PHONY: all build push rollout

all: build

login:
	aws ecr get-login-password --region ap-northeast-1 --profile jvd-demo \
	| docker login --username AWS --password-stdin ${ECR_REPO}

ssh-add:
	ssh-add ~/.ssh/id_rsa

build:
	docker buildx build \
		--platform ${PLATFORM} \
		--ssh default \
		-t ${ECR_REPO}/fatidentitycat .

push:
	docker push ${ECR_REPO}/fatidentitycat:latest

rollout:
	kubectl rollout restart deployment/fatidentitycat-web -n bonuscat
	kubectl rollout restart deployment/fatidentitycat-consumer -n bonuscat
	kubectl rollout restart deployment/fatidentitycat-worker -n bonuscat

