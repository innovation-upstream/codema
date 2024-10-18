# Codema: Streamlined API Development through Model-Driven Code Generation

## Introduction

Codema is a powerful code generation tool designed to revolutionize API development by enabling developers to define APIs in terms of their models, relationships, and specific functions. It then renders all non-model-specific code using standardized "implementation" templates. This approach offers immense benefits in terms of code safety, flexibility, and developer productivity.

### Key Benefits:

1. **Code Safety**: By separating model definitions from implementation details, Codema reduces the risk of inconsistencies and errors across different parts of your codebase.

2. **Code Flexibility**: Standardized templates allow for easy updates and modifications to implementation patterns across your entire project.

3. **Developer Productivity**: Codema automates the generation of boilerplate code, allowing developers to focus on core business logic and unique features.

4. **Consistency**: Ensures a uniform approach to API development across teams and projects.

5. **Easier Maintenance**: With clear separation of concerns, maintaining and updating your API becomes more straightforward.

## Core Concepts

### Target

A Target in Codema represents a specific output or destination for generated code. It defines where and how the code should be generated based on the API definitions.

Example:
```starlark
create_target(
    "handler",
    "/service/internal/rpc/handler",
    True,
    "v0.1.0",
    serviceInternalDir + "/rpc/handler.codema.go",
    apis
)
```

### API (to be deprecated)

An API in Codema represents a collection of related microservices. It's a high-level organizational unit.

### Microservice (to be deprecated)

A Microservice in Codema represents a specific service within an API, containing models and function implementations.

### Model

A Model in Codema defines the structure and properties of your data. It's the core building block of your API definitions.

Example:
```starlark
create_model(
    "User",
    [
        create_field("ID", "String", "The unique identifier for a user.", tags=[TAG_ID]),
        create_field("Name", "String", "The name of the user.", tags=[TAG_UPDATABLE]),
        create_field("Email", "String", "The email address of the user.", tags=[TAG_UPDATABLE]),
    ],
    "Represents a user in the system."
)
```

### FunctionImplementation

A FunctionImplementation in Codema defines how a specific function should be implemented across different targets.

Example:
```starlark
create_function_implementation(
    create_user_function,
    {
        "handler": "/snippets/handler/create.template",
        "logic": "/snippets/logic/create.template",
        "repo": "/snippets/repo/create.template",
    }
)
```

### Function

A Function in Codema represents a specific operation or action that can be performed within your API.

Example:
```starlark
create_function(
    "CreateUser",
    ["name", "email"],
    "Creates a new user in the system."
)
```

### Snippet

A Snippet in Codema is a template file that defines how a specific piece of code should be generated for a particular target.

### Hooks

Hooks in Codema allow you to inject custom logic at specific points in the code generation process, providing flexibility and customization.

## Core Use Case: Defining and Generating APIs

1. Define your API structure, microservices, and models:

```starlark
user_model = create_model(
    "User",
    [
        create_field("ID", "String", "The unique identifier for a user.", tags=[TAG_ID]),
        create_field("Name", "String", "The name of the user.", tags=[TAG_UPDATABLE]),
        create_field("Email", "String", "The email address of the user.", tags=[TAG_UPDATABLE]),
    ],
    "Represents a user in the system."
)

user_microservice = create_microservice(
    "user",
    user_model,
    [],
    [create_user_implementation, update_user_implementation, delete_user_implementation]
)

user_api = create_api(
    "user",
    "user",
    [user_microservice]
)
```

2. Define your targets:

```starlark
create_target(
    "handler",
    "/service/internal/rpc/handler",
    True,
    "v0.1.0",
    serviceInternalDir + "/rpc/handler.codema.go",
    [user_api]
)
```

3. Generate code:

```bash
codema generate -c starlark
```

This command will process your Starlark configuration, apply the defined models and function implementations to the specified targets, and generate the corresponding code.

## CLI Commands

### Generate

Generates code based on your API definitions.

```bash
codema generate [-t targets] [-c config_format]
```

- `-t, --targets`: Specify which targets to render (default is all)
- `-c, --config`: Specify the configuration format (yaml or starlark)

### Pull

Pulls pattern updates from a remote repository.

```bash
codema pull [patternLabel]
```

### Init

Initializes a new Codema pattern.

```bash
codema init [patternLabel]
```

### Publish

Publishes a Codema pattern.

```bash
codema publish [patternLabel]@[version]
```

## Example Codema Pattern Project

Here's an example of a simple Codema pattern project structure:

```
my-api-pattern/
├── codema-pattern.json
├── codema.star
├── models/
│   └── user.star
├── implementations/
│   ├── create_user.star
│   └── update_user.star
└── snippets/
    ├── handler/
    │   ├── create.template
    │   └── update.template
    └── repo/
        ├── create.template
        └── update.template
```

To publish this pattern:

1. Initialize the pattern:
   ```bash
   codema init my-api-pattern
   ```

2. Develop your models, implementations, and snippets.

3. Publish the pattern:
   ```bash
   codema publish my-api-pattern@1.0.0
   ```

This will package your pattern and make it available for others to use in their Codema projects.

By leveraging Codema's model-driven approach and standardized templates, you can significantly streamline your API development process, ensuring consistency, flexibility, and productivity across your projects.
