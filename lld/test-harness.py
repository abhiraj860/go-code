import sys
import traceback

try:
    from main import (
        LegacyPDFTool, HTMLWriter, PDFWriterAdapter,
        StandardWriterFactory, DocumentConversionFacade, DocumentWriter
    )
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def test_html_writer():
    writer = HTMLWriter()
    assert writer.write("Hello") == "<html>Hello</html>", "HTMLWriter output incorrect"

def test_adapter_delegation():
    legacy_tool = LegacyPDFTool()
    adapter = PDFWriterAdapter(legacy_tool)
    
    assert isinstance(adapter, DocumentWriter), "Adapter must implement DocumentWriter"
    result = adapter.write("Secret Report")
    assert result == "PDF[System]: Secret Report", f"Adapter output incorrect: {result}"

def test_factory_method_creation():
    factory = StandardWriterFactory()
    
    html_writer = factory.create_writer("html")
    assert isinstance(html_writer, HTMLWriter), "Factory should create HTMLWriter for 'html'"
    
    pdf_writer = factory.create_writer("pdf")
    assert isinstance(pdf_writer, PDFWriterAdapter), "Factory should create PDFWriterAdapter for 'pdf'"
    
    try:
        factory.create_writer("docx")
        assert False, "Factory should raise ValueError for unsupported format"
    except ValueError:
        pass

def test_facade_validation():
    factory = StandardWriterFactory()
    facade = DocumentConversionFacade(factory)
    
    try:
        facade.process_document("   ", "html")
        assert False, "Facade should raise ValueError for empty/whitespace content"
    except ValueError:
        pass

def test_end_to_end_integration():
    factory = StandardWriterFactory()
    facade = DocumentConversionFacade(factory)
    
    # Test through HTML path
    html_res = facade.process_document("My Page", "html")
    expected_html = "Facade processed: <html>My Page</html>"
    assert html_res == expected_html, f"Expected '{expected_html}', got '{html_res}'"
    
    # Test through Factory -> Adapter -> Legacy path
    pdf_res = facade.process_document("My Report", "pdf")
    expected_pdf = "Facade processed: PDF[System]: My Report"
    assert pdf_res == expected_pdf, f"Expected '{expected_pdf}', got '{pdf_res}'"

if __name__ == "__main__":
    tests = [
        test_html_writer,
        test_adapter_delegation,
        test_factory_method_creation,
        test_facade_validation,
        test_end_to_end_integration
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