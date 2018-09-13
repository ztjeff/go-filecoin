variable "name" {}
variable "hosted_zone_name" {}
variable "hosted_zone_id" {}
variable "vpc_id" {}
variable "public_subnets" { type = "list" }
variable "private_subnets" { type = "list" }
variable "container_definition" {}
variable "port" {}
variable "cpu" {}
variable "memory" {}
variable "min_capacity" {}
variable "max_capacity" {}
variable "execution_role_arn" {}
variable "instances_security_group_id" {}
variable "lb_target_group_arn" {}

# module "cert" {
#   source = "../route53/acm"

#   subdomain_name   = "${var.name}"
#   hosted_zone_name = "${var.hosted_zone_name}"
#   hosted_zone_id   = "${var.hosted_zone_id}"
# }

# module "alb" {
#   source = "../alb"

#   name = "${var.name}"

#   vpc_id           = "${var.vpc_id}"
#   public_subnets   = "${var.public_subnets}"
#   certificate_arn  = "${module.cert.certificate_arn}"
# }

# resource "aws_route53_record" "alb" {
#   zone_id = "${var.hosted_zone_id}"
#   name    = "${var.name}.${var.hosted_zone_name}"
#   type    = "A"

#   alias {
#     name    = "${module.alb.dns_name}"
#     zone_id = "${module.alb.zone_id}"

#     evaluate_target_health = true
#   }
# }

resource "aws_ecs_cluster" "this" {
  name = "${var.name}"
}

resource "aws_cloudwatch_log_group" "this" {
  name = "${var.name}"
}

module "service" {
  source = "./service"

  name                 = "${var.name}"
  container_definition = "${var.container_definition}"
  port                 = "${var.port}"
  cpu                  = "${var.cpu}"
  memory               = "${var.memory}"
  min_capacity         = "${var.min_capacity}"
  max_capacity         = "${var.max_capacity}"

  vpc_id          = "${var.vpc_id}"
  private_subnets = "${var.private_subnets}"

  cluster_id   = "${aws_ecs_cluster.this.id}"
  cluster_name = "${aws_ecs_cluster.this.name}"

  # alb_http_listener_arn  = "${module.alb.http_listener_arn}"
  # alb_https_listener_arn = "${module.alb.https_listener_arn}"
  instances_security_group_id  = "${var.instances_security_group_id}"
  lb_target_group_arn = "${var.lb_target_group_arn}"

  # dns_fqdn = "${aws_route53_record.alb.fqdn}"

  execution_role_arn = "${var.execution_role_arn}"
}
