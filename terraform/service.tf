variable "tabot_image" {
  type = string
}

resource "aws_ecs_task_definition" "tabot_task" {
  container_definitions = jsonencode([
    {
      name  = "tabot"
      image = var.tabot_image

      essential = true

      environment = [
        # TODO: Define
      ]

      secrets = [
        # TODO: Define
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = "microservices"
          awslogs-region        = data.aws_region.current.name
          awslogs-stream-prefix = "tabot"
        }
      }
    }
  ])

  cpu                      = 128
  memory                   = 128
  network_mode             = "awsvpc"
  family                   = "tabot"
  requires_compatibilities = ["FARGATE"]
  task_role_arn            = aws_iam_role.tabot_task_role.arn
  execution_role_arn       = aws_iam_role.tabot_execution_role.arn
}

resource "aws_ecs_service" "tabot-service-definition" {
  name            = "tabot"
  cluster         = data.aws_ssm_parameter.cluster.value
  task_definition = aws_ecs_task_definition.tabot_task.arn
  launch_type     = "FARGATE"
  desired_count   = 1
}