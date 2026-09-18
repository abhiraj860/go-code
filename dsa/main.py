class UserSession:
    active_sessions = 0
    def __init__(self, username: str, role: str, session_token: str) -> None:
        self.username = username
        self.role = role
        self.session_token = session_token
        type(self).active_sessions += 1
        
    @classmethod
    def create_guest(cls, username: str) -> "UserSession":
        return cls(username, "Guest", "GUEST-TEMP")
    
    @classmethod
    def create_admin(cls, username: str, session_token: str) -> "UserSession":
        return cls(username, "Admin", session_token)
    
    @staticmethod
    def validate_token(token: str) -> bool:
        return token.startswith("TOKEN-") and len(token) >= 8 
    
    
if __name__ == "__main__":
    # Test 1: Static Token Validation
    print(f"Token 'TOKEN-12345' Valid: {UserSession.validate_token('TOKEN-12345')}")
    print(f"Token 'SHORT' Valid: {UserSession.validate_token('SHORT')}")
    print(f"Token 'INVALID-123' Valid: {UserSession.validate_token('INVALID-123')}")

    # Test 2: Standard Constructor
    user1 = UserSession("alice_w", "Developer", "TOKEN-998877")
    print(f"\nUser: {user1.username} | Role: {user1.role} | Token: {user1.session_token}")

    # Test 3: Factory Methods
    guest = UserSession.create_guest("guest_bob")
    admin = UserSession.create_admin("boss_charlie", "TOKEN-554433")

    print(f"Guest: {guest.username} | Role: {guest.role} | Token: {guest.session_token}")
    print(f"Admin: {admin.username} | Role: {admin.role} | Token: {admin.session_token}")

    # Test 4: Global Counter Verification
    print(f"\nTotal Active Sessions: {UserSession.active_sessions}")
        