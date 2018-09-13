variable "name" {}
variable "vpc_id" {}
variable "private_subnets" { type = "list" }
variable "container_definition" {}
variable "port" {}
variable "cpu" {}
variable "memory" {}
variable "min_capacity" {}
variable "max_capacity" {}
variable "cluster_id" {}
variable "cluster_name" {}
variable "instances_security_group_id" {}
variable "lb_target_group_arn" {}
variable "execution_role_arn" {}

