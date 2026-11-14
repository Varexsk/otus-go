{{- define "calendar.dsn" -}}
postgres://{{ .Values.postgres.user }}:{{ .Values.postgres.password }}@{{ .Release.Name }}-postgres:5432/{{ .Values.postgres.database }}?sslmode=disable
{{- end -}}

{{- define "calendar.rabbitURL" -}}
amqp://{{ .Values.rabbitmq.user }}:{{ .Values.rabbitmq.password }}@{{ .Release.Name }}-rabbitmq:5672/
{{- end -}}

{{- define "calendar.secretName" -}}
{{ .Release.Name }}-secrets
{{- end -}}

{{- define "calendar.secretEnv" -}}
- name: {{ .name }}
  valueFrom:
    secretKeyRef:
      name: {{ include "calendar.secretName" .root }}
      key: {{ .key }}
{{- end -}}

{{- define "calendar.waitFor" -}}
- name: wait-for-{{ .host }}
  image: {{ .image }}
  imagePullPolicy: {{ .root.Values.imagePullPolicy }}
  command:
    - sh
    - -c
    - until timeout 3 nc {{ .host }} {{ .port }} </dev/null; do echo "waiting for {{ .host }}:{{ .port }}"; sleep 2; done
{{- end -}}
