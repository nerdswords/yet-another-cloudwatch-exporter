{
  'ebs.json': (import 'ebs.libsonnet') {
    uid: "aws-ebs",
  },
  'ec2.json': (import 'ec2.libsonnet') {
    uid: "aws-ec2",
  },
  'lambda.json': (import 'lambda.libsonnet') {
    uid: "aws-lambda",
  },
  'rds.json': (import 'rds.libsonnet') {
    uid: "aws-rds",
  },
  's3.json': (import 's3.libsonnet') {
    uid: "aws-s3",
  },
}
