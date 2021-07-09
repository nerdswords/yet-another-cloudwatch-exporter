output "cluster_name" {
  value = var.cluster_name
}

output "environment" {
  value = var.environment
}

output "namespace" {
  value = var.namespace
}

output "region" {
  value = var.region
}

output "aws_account_id" {
  value = data.aws_caller_identity.current.account_id
}

output "aws_policy_name" {
  value = local.policy_name
}

output "aws_role_name" {
  value = local.role_name
}

output "service_account_name" {
  value = var.service_account_name
}
