// This query identifies IAM Users and Roles that have inline policies with wildcard
// actions and resources, which could grant excessive permissions similar to AdministratorAccess.

// Find IAM Users with administrative inline policies
MATCH (user:`AWS::IAM::User`) 
WHERE user.userPolicyList IS NOT NULL 
  AND user.userPolicyList <> 'null'
  AND user.userPolicyList <> '[]'
  // Check for wildcard actions and resources in the same policy
  AND (
    // Full wildcard action and resource
    (user.userPolicyList CONTAINS '"Action":["*"]' AND user.userPolicyList CONTAINS '"Resource":["*"]') OR
    (user.userPolicyList CONTAINS '"Action":"*"' AND user.userPolicyList CONTAINS '"Resource":"*"') OR
    // IAM wildcard action with any wildcard resource (particularly dangerous)
    (user.userPolicyList CONTAINS '"Action":["iam:*"]' AND 
     (user.userPolicyList CONTAINS '"Resource":["*"]' OR user.userPolicyList CONTAINS 'arn:aws:iam::*')) OR
    // Service wildcard with a resource wildcard in the same statement
    (user.userPolicyList CONTAINS ':*"]' AND user.userPolicyList CONTAINS '"Resource":["*"]')
  )
RETURN 
    user.UserName AS Principal, 
    'User' AS PrincipalType, 
    user.arn AS ARN, 
    user.account AS AccountId,
    CASE 
        WHEN user.userPolicyList CONTAINS '"Action":["*"]' OR user.userPolicyList CONTAINS '"Action":"*"' 
        THEN 'Full Wildcard (*)'
        WHEN user.userPolicyList CONTAINS '"Action":["iam:*"]' 
        THEN 'IAM Service Wildcard (iam:*)'
        ELSE 'Service-specific Wildcard (service:*)'
    END AS ActionWildcardType,
    CASE 
        WHEN user.userPolicyList CONTAINS '"Resource":["*"]' OR user.userPolicyList CONTAINS '"Resource":"*"' 
        THEN 'Full Wildcard (*)'
        ELSE 'Partial Wildcard (arn:aws:*)'
    END AS ResourceWildcardType,
    user.userPolicyList AS InlinePolicies

UNION

// Find IAM Roles with administrative inline policies
MATCH (role:`AWS::IAM::Role`) 
WHERE role.rolePolicyList IS NOT NULL 
  AND role.rolePolicyList <> 'null'
  AND role.rolePolicyList <> '[]'
  // Check for wildcard actions and resources in the same policy
  AND (
    // Full wildcard action and resource
    (role.rolePolicyList CONTAINS '"Action":["*"]' AND role.rolePolicyList CONTAINS '"Resource":["*"]') OR
    (role.rolePolicyList CONTAINS '"Action":"*"' AND role.rolePolicyList CONTAINS '"Resource":"*"') OR
    // IAM wildcard action with any wildcard resource (particularly dangerous)
    (role.rolePolicyList CONTAINS '"Action":["iam:*"]' AND 
     (role.rolePolicyList CONTAINS '"Resource":["*"]' OR role.rolePolicyList CONTAINS 'arn:aws:iam::*')) OR
    // Service wildcard with a resource wildcard in the same statement
    (role.rolePolicyList CONTAINS ':*"]' AND role.rolePolicyList CONTAINS '"Resource":["*"]')
  )
RETURN 
    role.RoleName AS Principal, 
    'Role' AS PrincipalType, 
    role.arn AS ARN, 
    role.account AS AccountId,
    CASE 
        WHEN role.rolePolicyList CONTAINS '"Action":["*"]' OR role.rolePolicyList CONTAINS '"Action":"*"' 
        THEN 'Full Wildcard (*)'
        WHEN role.rolePolicyList CONTAINS '"Action":["iam:*"]' 
        THEN 'IAM Service Wildcard (iam:*)'
        ELSE 'Service-specific Wildcard (service:*)'
    END AS ActionWildcardType,
    CASE 
        WHEN role.rolePolicyList CONTAINS '"Resource":["*"]' OR role.rolePolicyList CONTAINS '"Resource":"*"' 
        THEN 'Full Wildcard (*)'
        ELSE 'Partial Wildcard (arn:aws:*)'
    END AS ResourceWildcardType,
    role.rolePolicyList AS InlinePolicies

// Order results by principal type and name
ORDER BY PrincipalType, Principal