INSERT INTO "_selfservice_registration_requests_tmp" (id, request_url, issued_at, expires_at, active_method, csrf_token, created_at, updated_at, messages) SELECT id, request_url, issued_at, expires_at, active_method, csrf_token, created_at, updated_at, messages FROM "selfservice_registration_requests"