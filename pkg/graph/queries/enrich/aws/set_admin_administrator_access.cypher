MATCH (p:Principal)
WHERE "arn:aws:iam::aws:policy/AdministratorAccess" IN p.attachedManagedPolicies
SET p.admin = true