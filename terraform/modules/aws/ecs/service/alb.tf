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
  listener_arn = "${var.alb_http_listener_arn}"

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
  listener_arn = "${var.alb_https_listener_arn}"

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
