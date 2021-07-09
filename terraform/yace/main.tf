provider "aws" {
  region = var.region
}

provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }
}

terraform {
  required_version = ">= 1.0"
}

resource "random_id" "id" {
  byte_length = 4
}

locals {
  account_id  = data.aws_caller_identity.current.account_id
  role_name   = "${var.environment}-yace_role-${random_id.id.hex}"
  policy_name = "${var.environment}-yace_policy-${random_id.id.hex}"
}

data "aws_caller_identity" "current" {}

data "aws_eks_cluster" "cluster" {
  name = var.cluster_name
}

# iam
resource "aws_iam_role" "yace_role" {
  name = local.role_name

  assume_role_policy = templatefile("${path.module}/templates/role.json", {
    account_id = local.account_id
    oidc       = replace(data.aws_eks_cluster.cluster.identity[0].oidc[0].issuer, "https://", "")
    namespace                = var.namespace
    service_account_username = var.service_account_name
  })
}

resource "aws_iam_policy" "yace_policy" {
  name   = local.policy_name
  policy = templatefile("${path.module}/templates/policy.json", {})
}

resource "aws_iam_role_policy_attachment" "yace_attach" {
  policy_arn = aws_iam_policy.yace_policy.arn
  role       = aws_iam_role.yace_role.name
}

resource "helm_release" "yace" {
  name       = "yace"

  chart      = var.chart
  namespace  = var.namespace

  set {
    name  = "yace.aws.account_id"
    value = data.aws_caller_identity.current.account_id
  }

  set {
    name  = "yace.aws.eks.cluster_name"
    value = var.cluster_name
  }

  set {
    name  = "yace.aws.role_name"
    value = aws_iam_role.yace_role.name
  }

  set {
    name  = "yace.serviceAccount"
    value = var.service_account_name
  }

  set {
    name  = "yace.config.region"
    value = var.region
  }
}
