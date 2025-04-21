// Find principals that can gain additional permissions through PassRole to EC2 instance profiles
MATCH (p:Principal)-[passRole:`iam:PassRole`]->(targetRole:Role)
WHERE 
  // Target role must have an instance profile
  targetRole.instanceProfileList IS NOT NULL 
  AND targetRole.instanceProfileList <> '[]'
WITH p, targetRole

// Principal must have RunInstances permissions
MATCH (p)-[runInstance:`ec2:RunInstances`]->()

// Get policies and permissions of principal and target role to compare
WITH p, targetRole, 
     CASE WHEN p.attachedManagedPolicies IS NULL THEN [] ELSE p.attachedManagedPolicies END AS principalPolicies,
     CASE WHEN targetRole.attachedManagedPolicies IS NULL THEN [] ELSE targetRole.attachedManagedPolicies END AS targetPolicies

// Check if target role has any policies not already attached to principal
WHERE (
  // Principal doesn't already have AdministratorAccess
  NOT ANY(policy IN principalPolicies WHERE policy CONTAINS 'AdministratorAccess')
  AND
  // Target role has policies not in principal's policies
  ANY(policy IN targetPolicies WHERE NOT policy IN principalPolicies)
)

RETURN DISTINCT 
  p.arn AS PrincipalARN, 
  CASE 
    WHEN p:User THEN 'User'
    WHEN p:Role THEN 'Role'
    ELSE labels(p)[0]
  END AS PrincipalType,
  CASE 
    WHEN p:User THEN p.userName
    WHEN p:Role THEN p.roleName
    ELSE NULL
  END AS PrincipalName,
  principalPolicies AS CurrentPolicies,
  targetRole.roleName AS TargetRoleName,
  targetPolicies AS TargetRolePolicies,
  [policy IN targetPolicies WHERE NOT policy IN principalPolicies] AS NewPoliciesGained,
  CASE 
    WHEN ANY(policy IN targetPolicies WHERE policy CONTAINS 'AdministratorAccess') 
    THEN true ELSE false 
  END AS CanGainAdminAccess
ORDER BY CanGainAdminAccess DESC, PrincipalType, PrincipalName