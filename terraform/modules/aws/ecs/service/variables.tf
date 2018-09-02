variable "name" {}
variable "vpc_id" {}
variable "private_subnets" { type = "list" }
variable "container_definition" {}
variable "port" {}
variable "health_check_url" {}
variable "cpu" {}
variable "memory" {}
variable "min_capacity" {}
variable "max_capacity" {}
variable "cluster_id" {}
variable "cluster_name" {}
variable "alb_http_listener_arn" {}
variable "alb_https_listener_arn" {}
variable "alb_security_group_id" {}
variable "dns_fqdn" {}
variable "execution_role_arn" {}
