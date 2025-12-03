# 設定環境變數，默認為 dev
ENV ?= dev

# 根據環境設定 ECR_REPO
ifeq ($(ENV),prod)
    ECR_REPO=REDACTED_AWS_ACCOUNT_PRO.dkr.ecr.ap-northeast-1.amazonaws.com/fatcat-extension
    NAMESPACE=fatcat-extension
    AWS_PROFILE=jvd-pro
    IMAGE_NAME=fat-identity-cat
else ifeq ($(ENV),dev)
    ECR_REPO=REDACTED_AWS_ACCOUNT_DEMO.dkr.ecr.ap-northeast-1.amazonaws.com/quantumcat
    NAMESPACE=bonuscat
    AWS_PROFILE=jvd-demo
    IMAGE_NAME=fatidentitycat
else
    $(error Invalid ENV value: $(ENV). Use 'dev' or 'prod')
endif

PLATFORM=linux/amd64
FULL_IMAGE=${ECR_REPO}/${IMAGE_NAME}:latest

.PHONY: all build push rollout deploy login ssh-add info

all: build

# 顯示當前環境資訊
info:
	@echo "=== 當前環境配置 ==="
	@echo "Environment: $(ENV)"
	@echo "ECR Repository: $(ECR_REPO)"
	@echo "Namespace: $(NAMESPACE)"
	@echo "AWS Profile: $(AWS_PROFILE)"
	@echo "Full Image: $(FULL_IMAGE)"
	@echo "====================="

login:
	@echo "Logging into ECR for $(ENV) environment..."
	aws ecr get-login-password --region ap-northeast-1 --profile $(AWS_PROFILE) \
	| docker login --username AWS --password-stdin $(ECR_REPO)

ssh-add:
	ssh-add ~/.ssh/id_rsa

build:
	@echo "Building image for $(ENV) environment..."
	docker buildx build \
		--platform $(PLATFORM) \
		--ssh default \
		-t $(FULL_IMAGE) .

push:
	@echo "Pushing image for $(ENV) environment..."
	docker push $(FULL_IMAGE)

rollout:
	@echo "Rolling out to $(ENV) environment (namespace: $(NAMESPACE))..."
	kubectl rollout restart deployment/fatidentitycat-web -n $(NAMESPACE)
	kubectl rollout restart deployment/fatidentitycat-consumer -n $(NAMESPACE)
	kubectl rollout restart deployment/fatidentitycat-worker -n $(NAMESPACE)
	kubectl rollout status deployment/fatidentitycat-web -n $(NAMESPACE) --timeout=300s
	kubectl rollout status deployment/fatidentitycat-consumer -n $(NAMESPACE) --timeout=300s
	kubectl rollout status deployment/fatidentitycat-worker -n $(NAMESPACE) --timeout=300s

