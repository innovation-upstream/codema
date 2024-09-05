# Utility functions
def create_field(name, type, description, optional=False, directives=None):
    return {
        "name": name,
        "type": type,
        "description": description,
        "optional": optional,
        "directives": directives if directives else {}
    }

def create_model(name, fields, description=""):
    return {
        "name": name,
        "fields": fields,
        "description": description
    }

def create_function(name, parameters, description=""):
    return {
        "name": name,
        "parameters": parameters,
        "description": description
    }

def create_function_implementation(function, target_snippets):
    return {
        "function": function,
        "target_snippets": target_snippets
    }

def create_microservice(label, models, function_implementations):
    return {
        "label": label,
        "models": models,
        "function_implementations": function_implementations,
    }

def create_api(package, label, microservices):
    return {
        "package": package,
        "label": label,
        "microservices": microservices
    }

# Define reusable directives
directives = {
    "updatable": {"updatable": True},
    "deprecated": {"deprecated": True},
}

# Define models
booking_model = create_model(
    "Booking",
    [
        create_field("id", "String", "The unique identifier for a booking.", optional=False),
        create_field("customer_id", "String", "Identifier of the customer who made the booking.", optional=False),
        create_field("start_time", "DateTime", "Start time of the booking.", optional=False),
        create_field("end_time", "DateTime", "End time of the booking.", optional=True),
        create_field("status", "BookingStatus", "Current status of the booking.", optional=False, directives=directives["deprecated"]),
        create_field("state", "BookingState", "The state of the booking.", optional=False, directives=directives["updatable"])
    ],
    "Represents a booking entity."
)

# Define functions
create_booking_function = create_function(
    "CreateBooking",
    ["customer_id", "start_time", "end_time"],
    "Creates a new booking."
)

update_booking_function = create_function(
    "UpdateBooking",
    ["booking_id", "start_time", "end_time", "state"],
    "Updates an existing booking."
)

# Define function implementations
create_booking_implementation = create_function_implementation(
    create_booking_function,
    {
        "repo": "/snippets/create_booking_repo.template",
        "grpc": "/snippets/create_booking_grpc.template"
    }
)

update_booking_implementation = create_function_implementation(
    update_booking_function,
    {
        "repo": "/snippets/update_booking_repo.template",
        "grpc": "/snippets/update_booking_grpc.template"
    }
)

# Define microservices
booking_microservice = create_microservice(
    "booking",
    [booking_model],
    [create_booking_implementation, update_booking_implementation],
)

booking_oracle_microservice = create_microservice(
    "bookingOracle",
    [],
    [create_booking_implementation],
)

# Define APIs
booking_api = create_api(
    "booking",
    "booking",
    [booking_microservice, booking_oracle_microservice]
)

service_provider_liability_model = create_model(
    "ServiceProviderLiability",
    [
        create_field("id", "String", "The unique identifier for a liability.", optional=False),
        create_field("service_provider_id", "String", "Identifier of the service provider.", optional=False),
        create_field("accounting_record_id", "String", "Identifier of the associated accounting record.", optional=False),
        create_field("amount", "Int64", "The liability amount.", optional=False),
        create_field("currency", "String", "The currency of the liability.", optional=False),
        create_field("created_at", "Timestamp", "Creation timestamp.", optional=False),
        create_field("updated_at", "Timestamp", "Last update timestamp.", optional=False),
        create_field("is_settled", "Bool", "Whether the liability is settled.", optional=False),
    ],
    "Represents a service provider liability."
)

# Define functions
svc_create_function = create_function(
    "Create",
    ["subjectID", "liability"],
    "Creates a new service provider liability."
)

update_function = create_function(
    "Update",
    ["subjectID", "id", "liability"],
    "Updates an existing service provider liability."
)

delete_function = create_function(
    "Delete",
    ["subjectID", "id"],
    "Deletes a service provider liability."
)

query_function = create_function(
    "Query",
    ["serviceProviderIDs", "accountingRecordIDs", "isSettled"],
    "Queries service provider liabilities based on given criteria."
)

# Define function implementations
create_implementation = create_function_implementation(
    svc_create_function,
    {
        "handler": "/snippets/handler/create.template",
        "logic": "/snippets/logic/create.template",
        "repo": "/snippets/repo/create.template",
        "relay": "/snippets/relay/create.template",
    }
)

update_implementation = create_function_implementation(
    update_function,
    {
        "handler": "/snippets/handler/update.template",
        "logic": "/snippets/logic/update.template",
        "repo": "/snippets/repo/update.template",
        "relay": "/snippets/relay/update.template",
    }
)

delete_implementation = create_function_implementation(
    delete_function,
    {
        "handler": "/snippets/handler/delete.template",
        "logic": "/snippets/logic/delete.template",
        "repo": "/snippets/repo/delete.template",
        "relay": "/snippets/relay/delete.template",
    }
)

query_implementation = create_function_implementation(
    query_function,
    {
        "handler": "/snippets/handler/query.template",
        "logic": "/snippets/logic/query.template",
        "repo": "/snippets/repo/query.template",
        "relay": "/snippets/relay/query.template",
    }
)

service_provider_liability_microservice = create_microservice(
    "serviceProviderLiability",
    [service_provider_liability_model],
    [
        create_implementation,
        update_implementation,
        delete_implementation,
        query_implementation,
    ],
)

# Define APIs
service_provider_liability_api = create_api(
    "service-provider",
    "serviceProviderLiability",
    [service_provider_liability_microservice]
)

# Configuration
config = {
    "moduleDir": "./out",
    "templateDir": "./codema-templates",
    "apis": [booking_api, service_provider_liability_api],
    "targets": [
        {
            "label": "relay-fn",
            "templateDir": "/relay-fn",
            "each": True,
            "defaultVersion": "v0.1.0",
            "apis": [
                {"label": "booking", "outPath": "/{{.Label}}/{{.Microservice.Label}}_relay_fn_generated.go"}
            ]
        },
        {
            "label": "grpc",
            "templatePath": "/grpc.template",
            "apis": [
                {"label": "booking", "outPath": "/{{.Label}}/grpc_generated.go"}
            ]
        },
        {
            "label": "repo",
            "templatePath": "/repo.template",
            "each": True,
            "apis": [
                {
                    "label": "booking",
                    "outPath": "/{{.Api.LabelKebab}}/{{.Microservice.Label}}_repo.codema.go",
                    "skipLabels": ["booking", "bookingOracle", "eagerBooking"]
                }
            ]
        },
          {
            "label": "handler",
            "templateDir": "/service/internal/rpc/handler",
            "each": True,
            "defaultVersion": "v0.1.0",
            "apis": [
                {"label": "serviceProviderLiability", "outPath": "/{{.Api.LabelKebab}}/{{.Microservice.LabelKebab}}/internal/rpc/handler_generated.go"}
            ]
        },
        {
            "label": "logic",
            "templateDir": "/service/internal/logic/logic",
            "each": True,
            "defaultVersion": "v0.1.0",
            "apis": [
                {"label": "serviceProviderLiability", "outPath": "/{{.Api.LabelKebab}}/{{.Microservice.LabelKebab}}/internal/logic/logic_generated.go"}
            ]
        },
        {
            "label": "repo",
            "templateDir": "/service/internal/repo/repo",
            "each": True,
            "defaultVersion": "v0.1.0",
            "apis": [
                {"label": "serviceProviderLiability", "outPath": "/{{.Api.LabelKebab}}/{{.Microservice.LabelKebab}}/internal/repo/repo_generated.go"}
            ]
        },
        {
            "label": "relay",
            "templateDir": "/service/external/relay/relay",
            "each": True,
            "defaultVersion": "v0.1.0",
            "apis": [
                {"label": "serviceProviderLiability", "outPath": "/{{.Api.LabelKebab}}/{{.Microservice.LabelKebab}}/internal/relay/relay_generated.go"}
            ]
        }
    ]
}
