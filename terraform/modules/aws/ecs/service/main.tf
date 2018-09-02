resource "aws_ecs_task_definition" "service" {
  family                   = "${var.name}"
  container_definitions    = "${var.container_definition}"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  task_role_arn            = "${aws_iam_role.ecs_task_role.arn}"
  execution_role_arn       = "${var.execution_role_arn}"
  cpu                      = "${var.cpu}"
  memory                   = "${var.memory}"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "service" {
  vpc_id      = "${var.vpc_id}"
  name        = "${var.name}-ecs-service-sg"
  description = "Allow egress from container"

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = "${var.port}"
    to_port     = "${var.port}"
    protocol    = "tcp"
    security_groups = ["${var.alb_security_group_id}"]
  }

  tags {
    Name        = "${var.name}-ecs-service-sg"
  }
}

resource "aws_ecs_service" "service" {
  name            = "${var.name}"
  task_definition = "${aws_ecs_task_definition.service.arn}"
  desired_count   = "${var.min_capacity}"
  launch_type     = "FARGATE"
  cluster         = "${var.cluster_id}"

  network_configuration {
    security_groups = ["${aws_security_group.service.id}"]
    subnets         = ["${var.private_subnets}"]
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = "${aws_alb_target_group.service.arn}"
    container_name   = "${var.name}"
    container_port   = "${var.port}"
  }

  # comment this block out if you want to restart the service with changes
  # to the local container_definitions for this service.
  # lifecycle {
  #   ignore_changes = ["task_definition"]
  # }

  depends_on = ["aws_alb_target_group.service"]
}
