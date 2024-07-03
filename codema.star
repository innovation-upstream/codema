# Define a function to create a field with metadata
def create_field(name, type, description, optional=False, directives=None):
    return {
        "name": name,
        "type": type,
        "description": description,
        "optional": optional,
        "directives": directives if directives else {}
    }

# Define models for the booking microservice
booking_models = {
    "Booking": {
        "fields": [
            create_field("id", "String", "The unique identifier for a booking.", optional=False),
            create_field("customer_id", "String", "Identifier of the customer who made the booking.", optional=False),
            create_field("start_time", "DateTime", "Start time of the booking.", optional=False),
            create_field("end_time", "DateTime", "End time of the booking.", optional=True),
            create_field("status", "BookingStatus", "Current status of the booking.", optional=False, directives={"deprecated": "Use 'state' field instead."}),
            create_field("state", "BookingState", "The state of the booking.", optional=False)
        ],
        "description": "Represents a booking entity."
    }
}

# Original configuration with models integrated
config = {
    "moduleDir": "./out",
    "templateDir": "./codema-templates",
    "apis": [
        {
            "package": "booking",
            "label": "booking",
            "microservices": [
                {"label": "booking", "models": booking_models["Booking"]},
                {"label": "bookingOracle", "models": {}},  # Assume no models defined for simplicity
                {"label": "bookingAttribute", "models": {}},  # Assume no models defined for simplicity
                {"label": "eagerBooking", "models": booking_models["Booking"]}  # Reusing the Booking model
            ]
        }
        # Add other packages following the same pattern
    ],
    "targets": [
      {
          "label": "relay",
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
                  "skipLabels": ["booking", "bookingOracle", "eagerBooking"],
                  "args": {
                      "bookingAttribute": {
                          "UpdateableFieldsBsonToGoFieldMap": {
                              "participant_owner_uids": "ParticipantOwnerUIDs"
                          }
                      }
                  }
              }
          ]
      }
  ]
}

