import sys
import traceback

try:
    from main import (
        EmailNotificationService, SMSNotificationService,
        EmailNotification, SMSNotification, Notification
    )
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def test_email_factory_creates_email_notification():
    service = EmailNotificationService()
    notif = service.create_notification()
    assert isinstance(notif, EmailNotification), "Factory should create EmailNotification"
    assert isinstance(notif, Notification), "EmailNotification should inherit from Notification"

def test_sms_factory_creates_sms_notification():
    service = SMSNotificationService()
    notif = service.create_notification()
    assert isinstance(notif, SMSNotification), "Factory should create SMSNotification"
    assert isinstance(notif, Notification), "SMSNotification should inherit from Notification"

def test_email_notification_send_format():
    service = EmailNotificationService()
    res = service.send_notification("Welcome aboard!", "alice@example.com")
    assert res == "Email sent to alice@example.com: Welcome aboard!", f"Unexpected format: {res}"

def test_sms_notification_send_format():
    service = SMSNotificationService()
    res = service.send_notification("Your OTP is 1234", "+15550199")
    assert res == "SMS sent to +15550199: Your OTP is 1234", f"Unexpected format: {res}"

def test_empty_recipient_raises_value_error():
    service = EmailNotificationService()
    try:
        service.send_notification("Hello", "  ")
        assert False, "Should raise ValueError for empty/whitespace recipient"
    except ValueError:
        pass

def test_empty_message_raises_value_error():
    service = SMSNotificationService()
    try:
        service.send_notification("", "+15550199")
        assert False, "Should raise ValueError for empty message"
    except ValueError:
        pass

if __name__ == "__main__":
    tests = [
        test_email_factory_creates_email_notification,
        test_sms_factory_creates_sms_notification,
        test_email_notification_send_format,
        test_sms_notification_send_format,
        test_empty_recipient_raises_value_error,
        test_empty_message_raises_value_error
    ]
    
    passed = 0
    for test in tests:
        try:
            test()
            print(f"PASS: {test.__name__}")
            passed += 1
        except Exception as e:
            print(f"FAIL: {test.__name__}")
            print(f"      {type(e).__name__}: {e}")
            traceback.print_exc(limit=-1, file=sys.stdout)
            
    print(f"\nSummary: {passed}/{len(tests)} passed.")
    if passed < len(tests):
        sys.exit(1)
    else:
        sys.exit(0)