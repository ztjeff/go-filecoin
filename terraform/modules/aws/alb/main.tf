variable "name" {}
variable "vpc_id" {}
variable "public_subnets" { type = "list" }
variable "certificate_arn" {}

resource "aws_alb" "alb" {
  name            = "${var.name}-alb"
  subnets         = ["${var.public_subnets}"]
  security_groups = ["${aws_security_group.alb.id}"]

  tags {
    Name = "${var.name}-alb"
  }
}

resource "aws_security_group" "alb" {
  name        = "${var.name}-alb-sg"
  description = "Allow HTTP/HTTPS from Anywhere into ALB"
  vpc_id      = "${var.vpc_id}"

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "${var.name}-alb-sg"
  }
}

# resource "random_id" "target_group" {
#   byte_length = 2
# }

# /* default target group and listeners */
# resource "aws_alb_target_group" "default" {
#   name                  = "${var.name}-alb-${random_id.target_group.hex}"
#   port                  = 80
#   protocol              = "HTTP"
#   vpc_id                = "${var.vpc_id}"
#   deregistration_delay  = 30

#   lifecycle {
#     create_before_destroy = true
#   }
# }

resource "aws_alb_listener" "http" {
  load_balancer_arn = "${aws_alb.alb.arn}"
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = "forward"
    target_group_arn = "${aws_alb_target_group.service.id}"
  }
}

resource "aws_alb_listener" "https" {
  load_balancer_arn = "${aws_alb.alb.arn}"
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
  certificate_arn   = "${var.certificate_arn}"

  default_action {
    type = "forward"
    target_group_arn = "${aws_alb_target_group.service.id}"
  }
}

# not needed since it' declared in aw_alb_listener.https ?
# resource "aws_alb_listener_certificate" "cert" {
#   listener_arn    = "${aws_alb_listener.https.arn}"
#   certificate_arn = "${var.certificate_arn}"
# }

################
resource "random_id" "target_group" {
  byte_length = 2
}

resource "aws_alb_target_group" "service" {
  name        = "${var.name}-alb-tg-${random_id.target_group.hex}"
  port        = "${var.port}"
  protocol    = "HTTP"
  vpc_id      = "${var.vpc_id}"
  target_type = "ip"

  health_check {
    path = "${var.health_check_url}"
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_alb_listener_rule" "http" {
  listener_arn = "${aws_alb_listener.http.arn}"

  action {
    type             = "forward"
    target_group_arn = "${aws_alb_target_group.service.arn}"
  }

  condition {
    field  = "host-header"
    values = ["${var.dns_fqdn}"]
  }

  depends_on = ["aws_alb_target_group.service"]
}

resource "aws_alb_listener_rule" "https" {
  listener_arn = "${aws_alb_listener.https.arn}"

  action {
    type             = "forward"
    target_group_arn = "${aws_alb_target_group.service.arn}"
  }

  condition {
    field  = "host-header"
    values = ["${var.dns_fqdn}"]
  }

  depends_on = ["aws_alb_target_group.service"]
}
########################

output "security_group_id" {
  value = "${aws_security_group.alb.id}"
}

output "dns_name" {
  value = "${aws_alb.alb.dns_name}"
}

output "zone_id" {
  value = "${aws_alb.alb.zone_id}"
}

output "default_target_group_arn" {
  value = "${aws_alb_target_group.default.arn}"
}

output "http_listener_arn" {
  value = "${aws_alb_listener.http.arn}"
}

output "https_listener_arn" {
  value = "${aws_alb_listener.https.arn}"
}

