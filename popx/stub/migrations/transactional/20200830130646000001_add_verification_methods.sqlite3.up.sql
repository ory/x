INSERT INTO "_selfservice_verification_flows_tmp" (id, request_url, issued_at, expires_at, via, csrf_token, success, created_at, updated_at, messages, type, state, active_method) SELECT id, request_url, issued_at, expires_at, via, csrf_token, success, created_at, updated_at, messages, type, state, active_method FROM "selfservice_verification_flows"