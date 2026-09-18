class EmployeeProfile:
    def __init__(self, name: str, ssn: str, salary: float):
        self.name = name
        self.__ssn = ssn
        self.__salary = salary

    @property
    def salary(self) -> float:
        return self.__salary
    
    @salary.setter
    def salary(self, amount: float) -> None:
        if amount <= 0.0:
            raise ValueError("Salary must be positive.")
        self.__salary = amount
    
    @property
    def masked_ssn(self) -> str:
        return "XXX-XX-" + self.__ssn[-4:]
    
    
if __name__ == "__main__":
    emp = EmployeeProfile("Sarah Connor", "123-45-6789", 75000.0)

    # Test 1: Property Getters
    print(f"Employee: {emp.name}")
    print(f"Masked SSN: {emp.masked_ssn}")
    print(f"Current Salary: ${emp.salary:.2f}")

    # Test 2: Valid Salary Update
    emp.salary = 82000.0
    print(f"Updated Salary: ${emp.salary:.2f}")

    # Test 3: Invalid Salary Validation Safeguard
    try:
        emp.salary = -5000.0
    except ValueError as e:
        print(f"PASS: Caught invalid salary update -> {e}")

    # Test 4: Private Name-Mangling Access Verification
    print(f"Mangled Salary Access: ${emp._EmployeeProfile__salary:.2f}") 