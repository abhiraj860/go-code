from abc import ABC, abstractmethod

# --- THE LEGACY TOOL (Adaptee - DO NOT MODIFY) ---
class LegacyPDFTool:
    def generate_pdf(self, content: str, metadata: dict) -> str:
        return f"PDF[{metadata.get('author', 'Unknown')}]: {content}"

# --- STANDARD INTERFACE ---
class DocumentWriter(ABC):
    @abstractmethod
    def write(self, text: str) -> str:
        pass

class HTMLWriter(DocumentWriter):
    def write(self, text: str) -> str:
        return f"<html>{text}</html>"

# --- ADAPTER ---
class PDFWriterAdapter(DocumentWriter):
    def __init__(self, pdf_tool: LegacyPDFTool):
        self.pdf_tool = pdf_tool
        
    def write(self, text: str) -> str:
        return self.pdf_tool.generate_pdf(text, {"author": "System"})        

# --- FACTORY METHOD ---
class WriterFactory(ABC):
    @abstractmethod
    def create_writer(self, format_type: str) -> DocumentWriter:
        pass

class StandardWriterFactory(WriterFactory):
    def create_writer(self, format_type: str) -> DocumentWriter:
        if format_type == "html":
            return HTMLWriter()
        elif format_type == "pdf":
            return PDFWriterAdapter(LegacyPDFTool())
        else:
            raise ValueError("Unsupported format")

# --- FACADE ---
class DocumentConversionFacade:
    def __init__(self, factory: WriterFactory):
        self.factory = factory

    def process_document(self, content: str, format_type: str) -> str:
        if content.strip() == "":
            raise ValueError("Empty whitespace")
        writer = self.factory.create_writer(format_type).write(content)
        return "Facade processed: " + writer