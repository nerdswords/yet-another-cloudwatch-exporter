variable "environment" {
  type = string

  validation {
    condition     = can(regex("^(demo|dev|qa|prod)$", var.environment))
    error_message = "The environment can be either dev/demo/qa/prod."
  }
}

variable "region" {
  type = string

  validation {
    condition     = can(regex("^(us|af|ap|ca|eu|me|sa)\\-(east|west|south|northeast|southeast|central|north)\\-(1|2|3)$", var.region))
    error_message = "The region must be a proper AWS region."
  }
}

# helm
variable "cluster_name" {
  type = string
}

variable "namespace" {
  type    = string
  default = "default"
}

variable "chart" {
  type = string
}

variable "service_account_name" {
  type    = string
  default = "yace-sa"
}
