# In-Memory File System: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a file and a directory are both "nodes", so size, search, delete and move treat them the same way, and a directory's size is just the sum of its children. A node's name is stored in exactly one place, as the key under which its parent holds it, so a rename is one change. Paths are plain values that are cleaned up (`.` and `..` removed) before anything walks the tree from the root, so no node has to point back to its parent.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The file system should organize files and directories in a tree under a single root directory.
2. Users should be able to create, delete, rename and move files and directories.
3. A directory should be able to contain both files and other directories, and both should be treated the same way where possible.
4. Users should be able to read, write and append to a file.
5. Users should be able to list the contents of a directory.
6. Users should be able to refer to items by absolute and relative paths.
7. Users should be able to search for files by name or extension.
8. The system should report the size of a file or of a whole directory.

**Types and conventions**
- A path is text such as `/docs/a.txt` (absolute) or `img/../a.txt` (relative to the current directory). `.` means "here" and `..` means "one level up". `..` at the root stays at the root.
- A name can't be empty, can't be `.` or `..`, and can't contain `/`.
- The size of a file is the number of characters in its content.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- No locks are used, because no requirement mentions concurrent use.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Path | Turns path text into clean, absolute steps from the root | the list of names |
| Node (abstract) | Is anything in the tree, and answers size and search | none |
| File | Holds text content | the content |
| Directory | Holds named children of either kind | its children and their names |
| EntryInfo | Gives a read-only picture of one listed item | name, kind, size |
| SearchCriterion (interface) | Decides whether a file name matches | none |
| NameEquals | Matches an exact file name | the name |
| ExtensionEquals | Matches a file extension | the extension |
| FileSystem | Resolves paths and runs every user action | the root, the current directory |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** File system, file, directory, root, tree, path, name, extension, content, size, current directory.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* File, Directory, Path and FileSystem are classes. Name, extension, content and size are values.
2. **Find fixed sets of values.** None. "File or directory" is the class itself.
3. **Give every fact one owner.** A file's content belongs to the File. A child's name belongs to its parent Directory, as the key it is stored under. The current directory belongs to the FileSystem.
   - *Why:* if a node also stored its own name, a rename would have to change two places.
4. **Mark what can be computed and do not store it.** A directory's size is the sum of its children's sizes. A file's full path is built while walking, never stored.
   - *Why:* a stored directory size would be wrong after the first write deep inside it.
5. **Give every verb to the class whose data changes.** Writing changes a File. Adding, removing and renaming children changes a Directory.
6. **Find the action that touches several classes.** A move needs the source's parent, the destination directory and a check against moving a directory into itself. That order lives in `FileSystem.move`.
7. **Find the hint phrases for patterns.** "Both should be treated the same way where possible" is Composite. "Search for files by name or extension" is a swappable criterion, which is Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** No requirement mentions it, so no locks.
10. **Fix the outputs.** A `bool` for every change, text or None for a read, lists for listing and search, an `int` or None for a size.

## 4. Pattern Guide

### Used: Composite
- **Meaning:** single items and groups of items share one interface, so code can treat a whole subtree like one item.
- **Hint phrase:** "a directory should be able to contain both files and other directories, and both should be treated the same way where possible".
- **How it shows here:** `get_size` and `find` work on any `Node`. A directory answers by asking its children, a file answers by itself.
- **When NOT to use it:** when the two kinds share almost nothing.

### Used: Strategy (search)
- **Meaning:** a rule is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrase:** "search for files by name or extension".
- **Without it:** search would have an `if` on the kind of search, and a new kind would mean editing it.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. |
| Observer | "notify when something changes" | Not used. |
| Factory | "create different kinds of objects by type" | Not used. The two create methods build each kind directly. |
| Facade | "a single simple entry point" | Not added as a separate class. `FileSystem` is already the single entry point. |
| Singleton | "exactly one instance" | Not used. |

## 5. Classes (smallest dependency first)

### Path
**Why this class exists:** requirement 6 accepts absolute and relative paths. Turning any path into clean steps from the root in one place means nothing else ever deals with `.`, `..` or the current directory.
- Attributes: `names: list[str]`. *Why:* the steps from the root, such as `["docs", "a.txt"]`. An empty list is the root itself.
- Constructor: `Path(names: list[str])`. *Why:* it keeps its own copy and refuses any name that is invalid. A Path never changes.
- Methods:
  - `parse(text: str, current: Path) -> Path or None` (a class method). *Why:* starts from the root if the text begins with `/`, otherwise from `current`. It skips empty parts and `.`, removes one step for each `..` (stopping at the root), and adds every other part. None if the text is empty or a part is invalid.
  - `is_root() -> bool`. *Why:* computed from an empty list.
  - `get_name() -> str or None`. *Why:* the last step. None for the root.
  - `parent() -> Path or None`. *Why:* the path without its last step. None for the root.
  - `child(name: str) -> Path`. *Why:* one more step. Search uses it to build paths while it walks.
  - `starts_with(other: Path) -> bool`. *Why:* True if `other`'s steps begin this path. This is how "moving a directory into itself" is caught.
  - `to_text() -> str`. *Why:* such as `/docs/a.txt`, or `/` for the root.
- *Extensibility:* none needed.

### Node (abstract)
**Why this class exists:** requirement 3 treats files and directories the same way where possible. Both answer the same two questions, so callers don't care which they hold.
- Attributes: none. *Why:* a node's name lives in its parent, not in the node.
- Methods:
  - `is_directory() -> bool` (abstract). *Why:* the one place the kind is told apart.
  - `get_size() -> int` (abstract). *Why:* requirement 8, for either kind.
  - `find(path: Path, criterion: SearchCriterion) -> list[str]` (abstract). *Why:* requirement 7, for either kind. `path` is where this node sits, passed down by the caller.
- *Extensibility:* a new kind of node (a link) is a new subclass.

### File (extends Node)
**Why this class exists:** requirement 4 reads and changes file content. The file owns its text.
- Attributes: `content: str`. *Why:* the single fact about a file.
- Constructor: `File(content: str)`. *Why:* refuses None. Empty text is allowed.
- Methods:
  - `is_directory() -> bool`. *Why:* False.
  - `read() -> str`. *Why:* the whole content.
  - `write(content: str) -> bool`. *Why:* replaces the content. False for None.
  - `append(content: str) -> bool`. *Why:* adds to the end. False for None.
  - `get_size() -> int`. *Why:* the number of characters, computed.
  - `find(path: Path, criterion: SearchCriterion) -> list[str]`. *Why:* `[path.to_text()]` if `criterion.matches(path.get_name())`, else an empty list.
- *Extensibility:* none needed.

### Directory (extends Node)
**Why this class exists:** requirement 3 lets a directory hold both kinds. It owns its children and their names, so a name is stored exactly once.
- Attributes: `children: map{str: Node}` (a child's name to the child). *Why:* the only home of names, and a lookup by name.
- Constructor: `Directory()`. *Why:* starts empty.
- Methods:
  - `is_directory() -> bool`. *Why:* True.
  - `get_child(name: str) -> Node or None`. *Why:* the lookup by name.
  - `add_child(name: str, node: Node) -> bool`. *Why:* False if the name is invalid or already used.
  - `remove_child(name: str) -> Node or None`. *Why:* takes the child out and returns it, so a move can put it somewhere else. None if not there.
  - `rename_child(old_name: str, new_name: str) -> bool`. *Why:* re-keys the child. False if `old_name` is missing, or `new_name` is invalid or used. The node itself does not change.
  - `list_names() -> list[str]`. *Why:* the names, sorted.
  - `is_empty() -> bool`. *Why:* computed.
  - `get_size() -> int`. *Why:* the sum of every child's `get_size()`, computed each time. This is the Composite in action.
  - `find(path: Path, criterion: SearchCriterion) -> list[str]`. *Why:* asks each child with `path.child(name)` and joins the answers. A directory itself never matches, because requirement 7 searches files.
- *Extensibility:* none needed.

### EntryInfo
**Why this class exists:** requirement 5 lists a directory. A read-only record per item means callers can show the list without reaching into the tree.
- Attributes: `name: str`, `is_directory: bool`, `size: int`. *Why:* what a listing shows.
- Constructor: `EntryInfo(name: str, is_directory: bool, size: int)`. *Why:* all set once.
- Methods: `get_name() -> str`, `is_directory() -> bool`, `get_size() -> int`. *Why:* read-only views.
- *Extensibility:* a date field is added here.

### SearchCriterion (interface)
**Why it is an interface:** requirement 7 searches by name or by extension, so the walk depends on one method and not on any kind of search.
- Methods: `matches(file_name: str) -> bool`.
- *Extensibility:* a new kind (a name pattern with `*`) is a new class.

### NameEquals and ExtensionEquals (implement SearchCriterion)
**Why these classes exist:** the two kinds named in requirement 7.
- `NameEquals(name: str)`: attribute `name: str`. `matches` is True if the file name equals it. It refuses an empty name.
- `ExtensionEquals(extension: str)`: attribute `extension: str`, stored without the dot and in lower case. `matches` is True if the file name ends with `.` plus the extension, ignoring case. It refuses an empty extension.
- *Extensibility:* none needed.

### FileSystem (the manager and only entry point)
**Why this class exists:** every user action starts from path text, which must be turned into a node, often with its parent, before anything is changed. It owns the root and the current directory.
- Attributes:
  - `root: Directory`. *Why:* requirement 1's single root.
  - `current: Path`. *Why:* where relative paths start. It is a Path, not a node, so it never goes stale.
- Constructor: `FileSystem()`. *Why:* an empty root, with the current directory at the root.
- Methods:
  - `change_directory(path_text: str) -> bool`. *Why:* False unless the path resolves to a directory.
  - `get_current_directory() -> str`. *Why:* read-only view.
  - `create_file(path_text: str, content: str) -> bool`, `create_directory(path_text: str) -> bool`. *Why:* the parent must already be a directory, and the name must be free.
  - `delete(path_text: str, recursive: bool) -> bool`. *Why:* removes a file or directory. A non-empty directory needs `recursive`. The root, and any directory containing the current directory, can't be deleted.
  - `rename(path_text: str, new_name: str) -> bool`. *Why:* re-keys the item in its parent. The current directory can't be renamed while you are inside it.
  - `move(source_text: str, destination_dir_text: str) -> bool`. *Why:* moves an item into another directory, keeping its name. Refused for the root, for a directory into itself or its own subtree, for a name already used at the destination, and for anything containing the current directory.
  - `read_file(path_text: str) -> str or None`, `write_file(path_text: str, content: str) -> bool`, `append_file(path_text: str, content: str) -> bool`. *Why:* requirement 4. The path must resolve to a file.
  - `list_directory(path_text: str or None) -> list[EntryInfo] or None`. *Why:* requirement 5. None means the current directory. Returns None if the path is not a directory. Sorted by name.
  - `search(path_text: str or None, criterion: SearchCriterion) -> list[str] or None`. *Why:* requirement 7. Full paths of matching files under that directory. None if it is not a directory.
  - `get_size(path_text: str) -> int or None`. *Why:* requirement 8, for either kind. None if nothing is there.
- Private helpers:
  - `resolve(path: Path) -> Node or None`. *Why:* walks from the root, one name at a time.
  - `resolve_parent(path: Path) -> Directory or None`. *Why:* the directory that holds the last step. None for the root or if it is not a directory.
- *Extensibility:* search kinds are passed in. New node kinds are new subclasses.

## 6. Relationships
- FileSystem **has** Directory (the root, created in the constructor).
- Directory **has** Node (children live and die inside their directory).
- FileSystem **uses** Path (the current directory is a value).
- File, Directory **extend** Node.
- NameEquals, ExtensionEquals **implement** SearchCriterion.
- There are no enums, so there are no enum lines.
- `EntryInfo` and `SearchCriterion` appear only as method outputs and inputs, so no attribute holds them and they have no line.

## 7. Interaction

### Layers
1. **Entry point and manager:** FileSystem.
2. **The tree:** Node, Directory, File.
3. **Paths:** Path.
4. **Rules passed in and plain data:** SearchCriterion with its two kinds, EntryInfo.

### Call chains
- **Any action with a path**
  1. `User -> FileSystem.<action>(path_text, ...)`.
  2. `FileSystem -> Path.parse(path_text, current)`: clean, absolute steps.
  3. `FileSystem.resolve` or `resolve_parent -> Directory.get_child(name)` for each step from the root.
  4. `FileSystem -> File` or `Directory` method: the owner makes the change.
- **Size and search**
  1. `User -> FileSystem.get_size(path_text)`, then `Node.get_size()`. A directory asks each child.
  2. `User -> FileSystem.search(path_text, criterion)`, then `Node.find(path, criterion)`. A directory asks each child, and a file asks `SearchCriterion.matches(name)`.
- **Move**
  1. `FileSystem -> Path.starts_with` (no moving into itself), then `Directory.remove_child(name)` on the old parent, then `Directory.add_child(name, node)` on the new one.

### Who may call whom
- Users call `FileSystem` only.
- `FileSystem` calls `Path`, `Directory` and `File`.
- `Directory` calls its children. `File` calls `SearchCriterion`.
- No node knows its parent or its name, so nothing calls back up the tree, and there are no cycles.

## 8. Flows

### Resolve a path
1. `Path.parse(text, current)` builds the clean steps. None means bad text.
2. `resolve` starts at `root` and calls `get_child(name)` for each step. If a step is missing, or a step before the last is a file, return None.

### Create
1. `create_file(path_text, content)`. Parse. `parent = resolve_parent(path)`. If None, return False.
2. `parent.add_child(path.get_name(), File(content))`. False if the name is used.
- **Failure:** nothing is created. The same steps with `Directory()` create a directory.

### Delete
1. `delete(path_text, recursive)`. Parse. If it is the root, or `current.starts_with(path)`, return False. *Why:* you can't delete what you are standing in.
2. `node = resolve(path)`. If None, return False. If it is a non-empty directory and `recursive` is False, return False.
3. `resolve_parent(path).remove_child(path.get_name())`.
- **Failure:** nothing is removed.

### Rename
1. `rename(path_text, new_name)`. Parse. If it is the root or contains the current directory, return False.
2. `resolve_parent(path).rename_child(path.get_name(), new_name)`.
- **Failure:** a missing item, an invalid name or a used name changes nothing.

### Move
1. `move(source_text, destination_dir_text)`. Parse both. If the source is the root or contains the current directory, return False.
2. `destination = resolve(destination path)`. If it is not a directory, return False.
3. If the destination path starts with the source path, return False. *Why:* a directory moved into its own subtree would cut itself off from the root.
4. If the destination already has a child with the source's name, return False.
5. `node = old_parent.remove_child(name)`, then `destination.add_child(name, node)`.
- **Failure and undo:** steps 1 to 4 check everything first, so step 5's add can't fail. If it ever did, the node would be added back to `old_parent`.

### Read, write, append
1. Resolve the path. If it is not a file, return None or False. Otherwise ask the file.

### List, search and size
1. `list_directory` resolves a directory and builds one `EntryInfo` per child, using `get_size()` for each.
2. `search` resolves a directory and calls `find(path, criterion)`, which returns the full paths of matching files.
3. `get_size` resolves any node and calls `get_size()`.
- **Failure:** a missing path, or the wrong kind, returns None.

### Concurrency
- No requirement mentions concurrent use, so no lock is added.
- If it were needed later, one read-write lock in `FileSystem` would be the simple answer: reads (read, list, search, size) share it, and changes take it alone.

## 9. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new kind of search (a name pattern, a size limit) | A new class implementing `SearchCriterion` | No. |
| A new kind of node (a symbolic link) | A new subclass of `Node` | No for the tree. Resolving a link would be a new step in `resolve`. |
| Creating missing parent directories at once | A `create_directory` option that creates each missing step | One new branch in `create_directory`. |
| Extra details in listings (dates) | New fields in `EntryInfo` and on `Node` | Additions only. |

## 10. Interview Notes

### Key decisions
1. **A name is stored only as its parent's key.**
   - *Why:* a rename changes one place, and a node and its parent can never disagree about its name.
   - *Rejected:* a `name` field on each node as well as the parent's map.
2. **No node points back to its parent.**
   - *Why:* no cycles, and moving a node changes only the two directories involved.
   - *Rejected:* a parent pointer on every node, which must be kept in step on every move.
3. **Paths are cleaned up before any walking.**
   - *Why:* `.`, `..` and relative paths are handled once, in `Path.parse`.
   - *Rejected:* each method handling `..` itself while walking.
4. **Size is computed by the tree (Composite).**
   - *Why:* it is always right, whatever changed deep inside.
   - *Rejected:* a stored size on each directory, updated on every write.
5. **The current directory is a Path, not a node.**
   - *Why:* it can't point at a node that was moved or deleted, and changes that would break it are refused.
   - *Rejected:* holding the current directory as a node.

### Edge cases handled
- `..` at the root stays at the root.
- A directory can't be moved into itself or its own subtree.
- The root can't be deleted, renamed or moved.
- A directory containing the current directory can't be deleted, renamed or moved.
- A non-empty directory needs `recursive` to be deleted.
- Creating inside a missing parent, or over an existing name, is refused.
- Reading or writing a directory, or listing a file, is refused.
- A search for an extension ignores case.

### Likely follow-up questions
1. **"Isn't computing a directory's size every time slow?"** For a big tree, yes. A cached size per directory, updated on every change along the path to the root, makes it instant, at the cost of a second copy of the data that must be kept in step.
2. **"How would you support permissions?"** Store an owner and permission bits on each `Node`, and check them in `FileSystem` before each action.
3. **"How would you make it safe for many users?"** See section 8: one read-write lock for the whole tree is simple. Finer locks per directory are faster but must be taken in a fixed order to avoid deadlock.

## 11. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Permissions and owners | See follow-up question 2. |
| Links | See the second row of section 10. |
| Dates and times on files | Not required. They would be fields on `Node`. |
| Binary content | Content is text. Bytes would replace `str` in `File`. |
| Moving the current directory along with a move | Refused instead. |
| Concurrent users | See follow-up question 3. |
| Saving to disk | It is in memory, as the requirement says. |

## 12. Requirements Met
1. A tree under one root: `FileSystem.root`, `Directory.children`.
2. Create, delete, rename, move: the matching `FileSystem` methods, with `Directory.add_child`, `remove_child` and `rename_child`.
3. Both kinds treated the same: `Node` with `get_size` and `find`, implemented by `File` and `Directory`.
4. Read, write, append: `File.read`, `write` and `append`, through `FileSystem`.
5. List a directory: `FileSystem.list_directory` and `EntryInfo`.
6. Absolute and relative paths: `Path.parse` and `FileSystem.current`.
7. Search by name or extension: `SearchCriterion` with `NameEquals` and `ExtensionEquals`, and `Node.find`.
8. Size of a file or directory: `Node.get_size`.

## 13. Requirement Fit Check

**Setup used in every example:** `create_directory("/docs")`, `create_file("/docs/a.txt", "hello")`, `create_directory("/docs/img")`, `create_file("/docs/img/cat.png", "1234")`, `create_file("/readme.md", "hi")`. The current directory is `/`.

1. **A tree under one root**
   - *Success:* `list_directory("/")` shows `docs` (a directory) and `readme.md` (a file).
   - *Failure:* `create_file("/missing/x.txt", "")` returns False because `/missing` does not exist.
2. **Create, delete, rename, move**
   - *Success:* `rename("/readme.md", "README.md")` returns True. `move("/docs/a.txt", "/docs/img")` makes `/docs/img/a.txt`. `delete("/docs/img/cat.png", False)` returns True.
   - *Failure:* `move("/docs", "/docs/img")` returns False because it is a move into its own subtree. `delete("/docs", False)` returns False because it is not empty. `delete("/", True)` returns False.
3. **Both kinds treated the same**
   - *Success:* `get_size`, `delete` and `move` all accept either a file path or a directory path.
   - *Failure:* none. Where the kinds really differ (reading content, listing), the wrong kind is refused, as in examples 4 and 5.
4. **Read, write, append**
   - *Success:* `write_file("/docs/a.txt", "hey")` then `read_file` gives `hey`. `append_file("/docs/a.txt", " there")` then `read_file` gives `hey there`.
   - *Failure:* `read_file("/docs")` returns None because it is a directory. `write_file("/docs/nope.txt", "x")` returns False because the file does not exist.
5. **List a directory**
   - *Success:* `list_directory("/docs")` returns `a.txt` (file, size 5) and `img` (directory, size 4).
   - *Failure:* `list_directory("/docs/a.txt")` returns None because it is a file.
6. **Absolute and relative paths**
   - *Success:* after `change_directory("/docs")`, `read_file("a.txt")` and `read_file("img/../a.txt")` both give `hello`, and `read_file("../readme.md")` gives `hi`.
   - *Failure:* `change_directory("a.txt")` returns False because it is a file. `change_directory("/..")` stays at `/`.
7. **Search by name or extension**
   - *Success:* `search("/", NameEquals("a.txt"))` returns [`/docs/a.txt`]. `search("/", ExtensionEquals("PNG"))` returns [`/docs/img/cat.png`].
   - *Failure:* `search("/", ExtensionEquals("pdf"))` returns an empty list.
8. **Size of a file or directory**
   - *Success:* `get_size("/docs/a.txt")` is 5. `get_size("/docs")` is 5 + 4 = 9. `get_size("/")` is 9 + 2 = 11.
   - *Failure:* `get_size("/nothing")` returns None.
