{{/* DevOps Agent Task Template */}}
---
key: {{.Key}}
title: {{.Title}}
epic: {{.Epic}}
feature: {{.Feature}}
agent: devops
status: todo
priority: {{.Priority}}
{{- if .DependsOn}}
depends_on: [{{join (quote .DependsOn) ", "}}]
{{- end}}
created_at: {{formatTime .CreatedAt}}
---

# Task: {{.Title}}

## Goal

{{if not (isEmpty .Description)}}{{.Description}}{{else}}[Describe the DevOps goal and infrastructure requirements]{{end}}

## Infrastructure Requirements

### Compute Resources

- [ ] VM/container specifications defined
- [ ] Scaling requirements identified
- [ ] Resource limits configured

### Storage Requirements

- [ ] Storage type and size determined
- [ ] Backup strategy defined
- [ ] Retention policies documented

### Network Requirements

- [ ] Network topology defined
- [ ] Security groups/firewall rules configured
- [ ] Load balancing requirements identified

## Deployment Configuration

### CI/CD Pipeline

- [ ] Build pipeline configured
- [ ] Test automation integrated
- [ ] Deployment stages defined
- [ ] Rollback strategy implemented

### Environment Configuration

- [ ] Development environment setup
- [ ] Staging environment setup
- [ ] Production environment setup
- [ ] Environment variables documented

## Monitoring & Observability

### Metrics

- [ ] Key metrics identified
- [ ] Dashboards created
- [ ] Alerting rules configured

### Logging

- [ ] Log aggregation configured
- [ ] Log retention policies set
- [ ] Log analysis tools integrated

### Health Checks

- [ ] Liveness probes configured
- [ ] Readiness probes configured
- [ ] Dependency health checks added

## Security Requirements

- [ ] Security scanning integrated
- [ ] Secrets management configured
- [ ] Access controls implemented
- [ ] Compliance requirements met

## Acceptance Criteria

- [ ] Infrastructure provisioned successfully
- [ ] Deployment pipeline functional
- [ ] Monitoring and alerting operational
- [ ] Security requirements satisfied
- [ ] Documentation complete

## Implementation Checklist

- [ ] Infrastructure as Code (IaC) written
- [ ] Configuration management setup
- [ ] Secrets securely stored
- [ ] Backup and recovery tested
- [ ] Disaster recovery plan documented

## Notes

- Follow infrastructure naming conventions
- Tag all resources appropriately
- Document all manual steps
- Implement cost optimization strategies
