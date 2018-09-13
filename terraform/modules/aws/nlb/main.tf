variable "name" {}
variable "vpc_id" {}
variable "public_subnets" { type = "list" }
variable "port" {}

resource "aws_lb" "nlb" {
  name               = "${var.name}-nlb"
  load_balancer_type = "network"  
  subnets            = ["${var.public_subnets}"]

  tags {
    Name = "${var.name}-nlb"
  }
}

# resource "aws_security_group" "nlb" {
#   name        = "${var.name}-nlb-sg"
#   description = "Allow ${var.port} from Anywhere into NLB"
#   vpc_id      = "${var.vpc_id}"

#   ingress {
#     from_port   = "${var.port}"
#     to_port     = "${var.port}"
#     protocol    = "tcp"
#     cidr_blocks = ["0.0.0.0/0"]
#   }

#   egress {
#     from_port   = 0
#     to_port     = 0
#     protocol    = "-1"
#     cidr_blocks = ["0.0.0.0/0"]
#   }

#   tags {
#     Name = "${var.name}-nlb-sg"
#   }
# }

resource "random_id" "target_group" {
  byte_length = 2
}

/* default target group and listeners */
resource "aws_lb_target_group" "default" {
  name                  = "${var.name}-nlb-${random_id.target_group.hex}"
  port                  = "${var.port}"
  protocol              = "TCP"
  vpc_id                = "${var.vpc_id}"
  deregistration_delay  = 30
  target_type           = "ip"  

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_lb_listener" "tcp" {
  load_balancer_arn = "${aws_lb.nlb.arn}"
  port              = "${var.port}"
  protocol          = "TCP"

  default_action {
    type = "forward"
    target_group_arn = "${aws_lb_target_group.default.id}"
  }
}

# output "security_group_id" {
#   value = "${aws_security_group.nlb.id}"
# }

output "dns_name" {
  value = "${aws_lb.nlb.dns_name}"
}

output "zone_id" {
  value = "${aws_lb.nlb.zone_id}"
}

output "default_target_group_arn" {
  value = "${aws_lb_target_group.default.arn}"
}

output "tcp_listener_arn" {
  value = "${aws_lb_listener.tcp.arn}"
}
