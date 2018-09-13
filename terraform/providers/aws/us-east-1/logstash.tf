locals {
  name = "logstash"
  hosted_zone_name = "${aws_route53_zone.kittyhawk.name}"  
  hosted_zone_id = "${aws_route53_zone.kittyhawk.zone_id}"
  port = "5044"  
}

module "logstash_ecs_execution_role" {
  source = "../../../modules/aws/ecs/execution_role/"  
}

# module "cert" {
#   source = "../../../modules/aws/route53/acm"  

#   subdomain_name   = "${var.name}"
#   hosted_zone_name = "${var.hosted_zone_name}"
#   hosted_zone_id   = "${var.hosted_zone_id}"
# }

# module "alb_test" {
#   source = "../../../modules/aws/nlb"  

#   name = "${local.name}"

#   # TODO locals   ?
#   vpc_id         = "${module.vpc.vpc_id}"  
#   public_subnets = "${module.vpc.public_subnets}"  
#   port           = "${local.port}"  
#   health_check_url = "/"  
# }

module "logstash_nlb" {
  source = "../../../modules/aws/nlb"  

  name = "${local.name}"

  # TODO locals   ?
  vpc_id         = "${module.vpc.vpc_id}"  
  public_subnets = "${module.vpc.public_subnets}"  
  port           = "${local.port}"  
}

resource "aws_route53_record" "logstash_nlb" {
  zone_id = "${local.hosted_zone_id}"
  name    = "${local.name}.${local.hosted_zone_name}"
  type    = "A"

  alias {
    name    = "${module.logstash_nlb.dns_name}"
    zone_id = "${module.logstash_nlb.zone_id}"

    evaluate_target_health = true
  }
}

module "ecs-logstash" {
  source = "../../../modules/aws/ecs"  
  name = "logstash"
  container_definition = "${file("logstash_container.json")}"  
  port = "5044"  
  cpu = 256
  memory = 1024
  min_capacity = 1
  max_capacity = 3  

  hosted_zone_name = "${aws_route53_zone.kittyhawk.name}"  
  hosted_zone_id = "${aws_route53_zone.kittyhawk.zone_id}"

  vpc_id = "${module.vpc.vpc_id}"  
  private_subnets = "${module.vpc.private_subnets}"  
  execution_role_arn = "${module.logstash_ecs_execution_role.role_arn}"
    
  instances_security_group_id  = "${aws_security_group.filecoin.id}"
  lb_target_group_arn = "${module.logstash_nlb.default_target_group_arn}"
}
