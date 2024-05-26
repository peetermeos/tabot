resource "aws_iam_role" "tabot_execution_role" {
  name                = "postgres_execution_role"
  managed_policy_arns = ["arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"]
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = ["ecs-tasks.amazonaws.com", "events.amazonaws.com"]
        }
      },
    ]
  })

  inline_policy {
    name = "tabot_execution_role_policy"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
      ]
    })
  }
}

resource "aws_iam_role" "tabot_task_role" {
  name = "tabot_task_role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = ["ecs-tasks.amazonaws.com", "events.amazonaws.com"]
        }
      },
    ]
  })

  inline_policy {
    name = "tabot_task_role_policy"
    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Action = [
            "logs:CreateLogStream",
            "logs:PutLogEvents",
            "logs:DescribeLogStreams"
          ]
          Effect   = "Allow"
          Resource = "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:microservices:*"
        }
      ]
    })
  }
}