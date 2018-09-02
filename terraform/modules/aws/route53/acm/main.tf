variable "subdomain_name" {}
variable "hosted_zone_name" {}
variable "hosted_zone_id" {}

resource "aws_acm_certificate" "cert" {
  domain_name       = "${var.subdomain_name}.${var.hosted_zone_name}"
  validation_method = "DNS"
}

resource "aws_route53_record" "cert_validation" {
  name    = "${aws_acm_certificate.cert.domain_validation_options.0.resource_record_name}"
  type    = "${aws_acm_certificate.cert.domain_validation_options.0.resource_record_type}"
  zone_id = "${var.hosted_zone_id}"
  records = ["${aws_acm_certificate.cert.domain_validation_options.0.resource_record_value}"]
  ttl     = 30
}

resource "aws_acm_certificate_validation" "cert" {
  certificate_arn = "${aws_acm_certificate.cert.arn}"

  validation_record_fqdns = ["${aws_route53_record.cert_validation.fqdn}"]
}

output "certificate_arn" {
  value = "${aws_acm_certificate.cert.arn}"
}
