INSERT INTO "_selfservice_verification_requests_tmp" (id, request_url, issued_at, expires_at, csrf_token, created_at, updated_at, messages, via, success) SELECT id, request_url, issued_at, expires_at, csrf_token, created_at, updated_at, messages, via, success FROM "selfservice_verification_requests"