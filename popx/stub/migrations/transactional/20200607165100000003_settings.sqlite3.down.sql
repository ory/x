INSERT INTO "_selfservice_settings_requests_tmp" (id, request_url, issued_at, expires_at, identity_id, created_at, updated_at, active_method, messages) SELECT id, request_url, issued_at, expires_at, identity_id, created_at, updated_at, active_method, messages FROM "selfservice_settings_requests"