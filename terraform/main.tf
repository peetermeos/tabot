terraform {
  backend "http" {}
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.32.0"
    }
  }

}

variable "gitlab_access_token" {
  type = string
}

variable "aws_access_key_id" {
  type = string
}

variable "aws_secret_access_key" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "stage" {
  type = string
}

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

provider "aws" {
  region     = var.aws_region
  access_key = var.aws_access_key_id
  secret_key = var.aws_secret_access_key
}

