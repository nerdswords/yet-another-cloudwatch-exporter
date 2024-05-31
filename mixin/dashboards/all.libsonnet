{
  'ebs.json': (import 'ebs.libsonnet') {
    uid: std.md5('ebs.libsonnet'),
  },
  'ec2.json': (import 'ec2.libsonnet') {
    uid: std.md5('ec2.libsonnet'),
  },
  'lambda.json': (import 'lambda.libsonnet') {
    uid: std.md5('lambda.libsonnet'),
  },
  'rds.json': (import 'rds.libsonnet') {
    uid: std.md5('rds.libsonnet'),
  },
  's3.json': (import 's3.libsonnet') {
    uid: std.md5('s3.libsonnet'),
  },
}
