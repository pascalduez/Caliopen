// Copyleft (ɔ) 2017 The Caliopen contributors.
// Use of this source code is governed by a GNU AFFERO GENERAL PUBLIC
// license (AGPL) that can be found in the LICENSE file.

package rest_api

import (
	obj "github.com/CaliOpen/CaliOpen/src/backend/defs/go-objects"
	"github.com/CaliOpen/CaliOpen/src/backend/interfaces/REST/go.server/operations/users"
	"github.com/CaliOpen/CaliOpen/src/backend/main/go.main"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/gin-gonic/gin.v1"

        "github.com/go-openapi/loads"
        "github.com/go-openapi/runtime/middleware"
        "github.com/go-openapi/runtime/middleware/untyped"
        "encoding/json"
)

var (
	server *REST_API
	caliop *caliopen.CaliopenFacilities
)

type (
	REST_API struct {
		config APIConfig
	}

	APIConfig struct {
		Host          string `mapstructure:"host"`
		Port          string `mapstructure:"port"`
		BackendConfig `mapstructure:"BackendConfig"`
	}

	BackendConfig struct {
		BackendName string          `mapstructure:"backend_name"`
		Settings    BackendSettings `mapstructure:"backend_settings"`
	}

	BackendSettings struct {
		Hosts       []string `mapstructure:"hosts"`
		Keyspace    string   `mapstructure:"keyspace"`
		Consistency uint16   `mapstructure:"consistency_level"`
	}
)

func InitializeServer(config APIConfig) error {
	server = new(REST_API)
	return server.initialize(config)
}

func (server *REST_API) initialize(config APIConfig) error {
	server.config = config

	//init Caliopen facility
	caliopenConfig := obj.CaliopenConfig{
		RESTstoreConfig: obj.RESTstoreConfig{
			BackendName: config.BackendName,
			Hosts:       config.BackendConfig.Settings.Hosts,
			Keyspace:    config.BackendConfig.Settings.Keyspace,
			Consistency: config.BackendConfig.Settings.Consistency,
		},
	}

	err := caliopen.Initialize(caliopenConfig)

	if err != nil {
		log.WithError(err).Fatal("Caliopen facilities initialization failed")
	}

	caliop = caliopen.Facilities

	return nil
}

func StartServer() error {
	return server.start()
}

func (server *REST_API) start() error {
	// Creates a gin router with default middleware:
	// logger and recovery (crash-free) middleware
	router := gin.Default()
	// adds our middlewares

        //configure swagger
        spec, err := loads.Analyzed(json.RawMessage([]byte(swaggerJSON)), "")
        if err != nil {
                return nil
        }
        swag_api := untyped.NewAPI(spec)

        swag_api.RegisterOperation("get", "/v2/username/isAvailable", server)
        swag_http_handler := middleware.Serve(spec, swag_api) // http handler interface = ServeHTTP(w http.ResponseWriter, req *http.Request)

        router.Use(func(ctx *gin.Context){
                swag_http_handler.ServeHTTP(ctx.Writer, ctx.Request )
        })
	// adds our routes and handlers

	api := router.Group("/api/v2")
	AddHandlers(api)

	// listens
	addr := server.config.Host + ":" + server.config.Port

	err = router.Run(addr)
	if err != nil {
		log.WithError(err).Info("unable to start gin server")
	}
	return err
}

func AddHandlers(api *gin.RouterGroup) {

	//users API
	u := api.Group("/users")
	u.POST("/", func(ctx *gin.Context) {
		users.Create(caliop, ctx)
	})
	u.GET("/:user_id", func(ctx *gin.Context) {
		users.Get(caliop, ctx)
	})

	//username API
	api.GET("/username/isAvailable", func(ctx *gin.Context) {
		users.IsAvailable(caliop, ctx)
	})
}

func (server *REST_API) Handle(interface{}) (interface{}, error){
        return struct{}{}, nil
}

var swaggerJSON = `
{
  "swagger": "2.0",
  "info": {
    "version": "0.0.2",
    "title": "Caliopen HTTP/REST API"
  },
  "schemes": [
    "http"
  ],
  "host": "localhost:3141",
  "basePath": "/api",
  "paths": {
    "/v1/authentications": {
      "post": {
        "description": "Returns an auth token to build basicAuth for the provided credentials",
        "tags": [
          "users"
        ],
        "security": [],
        "consumes": [
          "application/json"
        ],
        "parameters": [
          {
            "name": "authentication",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "properties": {
                "username": {
                  "type": "string"
                },
                "password": {
                  "type": "string"
                }
              },
              "required": [
                "username",
                "password"
              ],
              "additionalProperties": false
            }
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Successful authentication",
            "schema": {
              "type": "object",
              "properties": {
                "username": {
                  "type": "string"
                },
                "user_id": {
                  "type": "string",
                  "description": "the user_id makes the 'username' for basicAuth"
                },
                "tokens": {
                  "type": "object",
                  "properties": {
                    "access_token": {
                      "type": "string",
                      "description": "the access_token makes the 'password' for basicAuth"
                    },
                    "expires_in": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "expires_at": {
                      "type": "string"
                    },
                    "refresh_token": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          },
          "401": {
            "description": "Authentication error",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/v1/me": {
      "get": {
        "description": "Gets user + contact objects for current logged-in user\n",
        "tags": [
          "users"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "X-Caliopen-PI",
            "in": "header",
            "required": true,
            "description": "The PI range requested in form of 1;100",
            "type": "string",
            "default": "1;100"
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Successful response with json object",
            "schema": {
              "type": "object",
              "properties": {
                "contact": {
                  "type": "object",
                  "properties": {
                    "additional_name": {
                      "type": "string"
                    },
                    "addresses": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "address_id": {
                            "type": "string"
                          },
                          "city": {
                            "type": "string"
                          },
                          "country": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "label": {
                            "type": "string"
                          },
                          "postal_code": {
                            "type": "string"
                          },
                          "region": {
                            "type": "string"
                          },
                          "street": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "address_id",
                          "city"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "avatar": {
                      "type": "string"
                    },
                    "contact_id": {
                      "type": "string"
                    },
                    "date_insert": {
                      "type": "string",
                      "format": "date-time"
                    },
                    "date_update": {
                      "type": "string",
                      "format": "date-time"
                    },
                    "deleted": {
                      "type": "boolean"
                    },
                    "emails": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "email_id": {
                            "type": "string"
                          },
                          "address": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "label": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "email_id",
                          "address"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "family_name": {
                      "type": "string"
                    },
                    "given_name": {
                      "type": "string"
                    },
                    "groups": {
                      "type": "array",
                      "items": {
                        "type": "string"
                      }
                    },
                    "identities": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "infos": {
                                "type": "object"
                              },
                              "name": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "identity_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "identity_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "ims": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "address": {
                                "type": "string"
                              },
                              "is_primary": {
                                "type": "boolean"
                              },
                              "label": {
                                "type": "string"
                              },
                              "protocol": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "address"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "im_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "im_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "infos": {
                      "type": "object"
                    },
                    "name_prefix": {
                      "type": "string"
                    },
                    "name_suffix": {
                      "type": "string"
                    },
                    "organizations": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "department": {
                                "type": "string"
                              },
                              "is_primary": {
                                "type": "boolean"
                              },
                              "job_description": {
                                "type": "string"
                              },
                              "label": {
                                "type": "string"
                              },
                              "name": {
                                "type": "string"
                              },
                              "title": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "label",
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "deleted": {
                                "type": "boolean"
                              }
                            },
                            "required": [
                              "organization_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "phones": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "is_primary": {
                                "type": "boolean"
                              },
                              "number": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              },
                              "uri": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "number"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "phone_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "phone_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "privacy_features": {
                      "type": "object"
                    },
                    "privacy_index": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "public_keys": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "expire_date": {
                                "type": "string",
                                "format": "date-time"
                              },
                              "fingerprint": {
                                "type": "string"
                              },
                              "key": {
                                "type": "string"
                              },
                              "name": {
                                "type": "string"
                              },
                              "size": {
                                "type": "integer",
                                "format": "int32"
                              }
                            },
                            "required": [
                              "key",
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "date_insert": {
                                "type": "string",
                                "format": "date-time"
                              },
                              "date_update": {
                                "type": "string",
                                "format": "date-time"
                              }
                            },
                            "required": [
                              "date_insert"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "tags": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "type": {
                            "type": "string",
                            "enum": [
                              "user",
                              "system"
                            ]
                          },
                          "name": {
                            "type": "string"
                          },
                          "tag_id": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "name",
                          "type"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "title": {
                      "type": "string"
                    },
                    "user_id": {
                      "type": "string"
                    }
                  },
                  "required": [
                    "contact_id",
                    "user_id"
                  ],
                  "additionalProperties": false
                },
                "date_insert": {
                  "type": "string",
                  "format": "date-time"
                },
                "family_name": {
                  "type": "string"
                },
                "given_name": {
                  "type": "string"
                },
                "name": {
                  "type": "string"
                },
                "password": {
                  "type": "string"
                },
                "params": {
                  "type": "object"
                }
              },
              "additionalProperties": false
            }
          }
        }
      }
    },
    "/v1/users": {
      "post": {
        "description": "Create a new User with provided credentials",
        "tags": [
          "users"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "consumes": [
          "application/json"
        ],
        "parameters": [
          {
            "name": "user",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "properties": {
                "contact": {
                  "type": "object",
                  "properties": {
                    "additional_name": {
                      "type": "string"
                    },
                    "addresses": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "address_id": {
                            "type": "string"
                          },
                          "city": {
                            "type": "string"
                          },
                          "country": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "label": {
                            "type": "string"
                          },
                          "postal_code": {
                            "type": "string"
                          },
                          "region": {
                            "type": "string"
                          },
                          "street": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "address_id",
                          "city"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "avatar": {
                      "type": "string"
                    },
                    "contact_id": {
                      "type": "string"
                    },
                    "date_insert": {
                      "type": "string",
                      "format": "date-time"
                    },
                    "date_update": {
                      "type": "string",
                      "format": "date-time"
                    },
                    "deleted": {
                      "type": "boolean"
                    },
                    "emails": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "email_id": {
                            "type": "string"
                          },
                          "address": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "label": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "email_id",
                          "address"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "family_name": {
                      "type": "string"
                    },
                    "given_name": {
                      "type": "string"
                    },
                    "groups": {
                      "type": "array",
                      "items": {
                        "type": "string"
                      }
                    },
                    "identities": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "infos": {
                                "type": "object"
                              },
                              "name": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "identity_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "identity_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "ims": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "address": {
                                "type": "string"
                              },
                              "is_primary": {
                                "type": "boolean"
                              },
                              "label": {
                                "type": "string"
                              },
                              "protocol": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "address"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "im_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "im_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "infos": {
                      "type": "object"
                    },
                    "name_prefix": {
                      "type": "string"
                    },
                    "name_suffix": {
                      "type": "string"
                    },
                    "organizations": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "department": {
                                "type": "string"
                              },
                              "is_primary": {
                                "type": "boolean"
                              },
                              "job_description": {
                                "type": "string"
                              },
                              "label": {
                                "type": "string"
                              },
                              "name": {
                                "type": "string"
                              },
                              "title": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "label",
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "deleted": {
                                "type": "boolean"
                              }
                            },
                            "required": [
                              "organization_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "phones": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "is_primary": {
                                "type": "boolean"
                              },
                              "number": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              },
                              "uri": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "number"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "phone_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "phone_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "privacy_features": {
                      "type": "object"
                    },
                    "privacy_index": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "public_keys": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "expire_date": {
                                "type": "string",
                                "format": "date-time"
                              },
                              "fingerprint": {
                                "type": "string"
                              },
                              "key": {
                                "type": "string"
                              },
                              "name": {
                                "type": "string"
                              },
                              "size": {
                                "type": "integer",
                                "format": "int32"
                              }
                            },
                            "required": [
                              "key",
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "date_insert": {
                                "type": "string",
                                "format": "date-time"
                              },
                              "date_update": {
                                "type": "string",
                                "format": "date-time"
                              }
                            },
                            "required": [
                              "date_insert"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "tags": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "type": {
                            "type": "string",
                            "enum": [
                              "user",
                              "system"
                            ]
                          },
                          "name": {
                            "type": "string"
                          },
                          "tag_id": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "name",
                          "type"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "title": {
                      "type": "string"
                    },
                    "user_id": {
                      "type": "string"
                    }
                  },
                  "required": [
                    "contact_id",
                    "user_id"
                  ],
                  "additionalProperties": false
                },
                "date_insert": {
                  "type": "string",
                  "format": "date-time"
                },
                "family_name": {
                  "type": "string"
                },
                "given_name": {
                  "type": "string"
                },
                "name": {
                  "type": "string"
                },
                "password": {
                  "type": "string"
                },
                "params": {
                  "type": "object"
                }
              },
              "additionalProperties": false
            }
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "User creation completed",
            "schema": {
              "type": "object",
              "properties": {
                "location": {
                  "type": "string",
                  "description": "url to retrieve new user's infos at /users/{user_id}"
                }
              }
            }
          }
        }
      }
    },
    "/v1/users/{user_id}": {
      "get": {
        "description": "Retrieve contact infos",
        "tags": [
          "users"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "user_id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Successful response with json object",
            "schema": {
              "type": "object",
              "properties": {
                "contact": {
                  "type": "object",
                  "properties": {
                    "additional_name": {
                      "type": "string"
                    },
                    "addresses": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "address_id": {
                            "type": "string"
                          },
                          "city": {
                            "type": "string"
                          },
                          "country": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "label": {
                            "type": "string"
                          },
                          "postal_code": {
                            "type": "string"
                          },
                          "region": {
                            "type": "string"
                          },
                          "street": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "address_id",
                          "city"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "avatar": {
                      "type": "string"
                    },
                    "contact_id": {
                      "type": "string"
                    },
                    "date_insert": {
                      "type": "string",
                      "format": "date-time"
                    },
                    "date_update": {
                      "type": "string",
                      "format": "date-time"
                    },
                    "deleted": {
                      "type": "boolean"
                    },
                    "emails": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "email_id": {
                            "type": "string"
                          },
                          "address": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "label": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "email_id",
                          "address"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "family_name": {
                      "type": "string"
                    },
                    "given_name": {
                      "type": "string"
                    },
                    "groups": {
                      "type": "array",
                      "items": {
                        "type": "string"
                      }
                    },
                    "identities": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "infos": {
                                "type": "object"
                              },
                              "name": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "identity_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "identity_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "ims": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "address": {
                                "type": "string"
                              },
                              "is_primary": {
                                "type": "boolean"
                              },
                              "label": {
                                "type": "string"
                              },
                              "protocol": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "address"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "im_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "im_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "infos": {
                      "type": "object"
                    },
                    "name_prefix": {
                      "type": "string"
                    },
                    "name_suffix": {
                      "type": "string"
                    },
                    "organizations": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "department": {
                                "type": "string"
                              },
                              "is_primary": {
                                "type": "boolean"
                              },
                              "job_description": {
                                "type": "string"
                              },
                              "label": {
                                "type": "string"
                              },
                              "name": {
                                "type": "string"
                              },
                              "title": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "label",
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "deleted": {
                                "type": "boolean"
                              }
                            },
                            "required": [
                              "organization_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "phones": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "is_primary": {
                                "type": "boolean"
                              },
                              "number": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              },
                              "uri": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "number"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "phone_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "phone_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "privacy_features": {
                      "type": "object"
                    },
                    "privacy_index": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "public_keys": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "expire_date": {
                                "type": "string",
                                "format": "date-time"
                              },
                              "fingerprint": {
                                "type": "string"
                              },
                              "key": {
                                "type": "string"
                              },
                              "name": {
                                "type": "string"
                              },
                              "size": {
                                "type": "integer",
                                "format": "int32"
                              }
                            },
                            "required": [
                              "key",
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "date_insert": {
                                "type": "string",
                                "format": "date-time"
                              },
                              "date_update": {
                                "type": "string",
                                "format": "date-time"
                              }
                            },
                            "required": [
                              "date_insert"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "tags": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "type": {
                            "type": "string",
                            "enum": [
                              "user",
                              "system"
                            ]
                          },
                          "name": {
                            "type": "string"
                          },
                          "tag_id": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "name",
                          "type"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "title": {
                      "type": "string"
                    },
                    "user_id": {
                      "type": "string"
                    }
                  },
                  "required": [
                    "contact_id",
                    "user_id"
                  ],
                  "additionalProperties": false
                },
                "date_insert": {
                  "type": "string",
                  "format": "date-time"
                },
                "family_name": {
                  "type": "string"
                },
                "given_name": {
                  "type": "string"
                },
                "name": {
                  "type": "string"
                },
                "password": {
                  "type": "string"
                },
                "params": {
                  "type": "object"
                }
              },
              "additionalProperties": false
            }
          },
          "401": {
            "description": "Unauthorized access",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          }
        }
      },
      "delete": {
        "description": "Not Yet Implemented",
        "tags": [
          "users"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "user_id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "responses": {
          "default": {
            "description": "route not implemented, should raise an error"
          }
        }
      },
      "patch": {
        "description": "Not Yet Implemented",
        "tags": [
          "users"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "user_id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "responses": {
          "default": {
            "description": "route not implemented, should raise an error"
          }
        }
      }
    },
    "/v2/username/isAvailable": {
      "get": {
        "description": "Check if an username is available for creation within Caliopen instance",
        "tags": [
          "users",
          "username"
        ],
        "security": [],
        "parameters": [
          {
            "name": "username",
            "in": "query",
            "required": true,
            "type": "string"
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "availability state for requested username",
            "schema": {
              "type": "object",
              "properties": {
                "username": {
                  "type": "string"
                },
                "available": {
                  "type": "boolean"
                }
              },
              "required": [
                "username",
                "available"
              ]
            }
          },
          "400": {
            "description": "malform request (probably missing 'username' query param)",
            "schema": {
              "type": "object",
              "properties": {
                "username": {
                  "type": "string"
                },
                "available": {
                  "type": "boolean"
                }
              },
              "required": [
                "username",
                "available"
              ]
            }
          }
        }
      }
    },
    "/v1/contacts": {
      "get": {
        "description": "Returns contacts belonging to current user according to given parameters",
        "tags": [
          "contacts"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "X-Caliopen-PI",
            "in": "header",
            "required": true,
            "description": "The PI range requested in form of 1;100",
            "type": "string",
            "default": "1;100"
          },
          {
            "name": "limit",
            "in": "query",
            "required": false,
            "type": "integer",
            "description": "number of contacts to return per page"
          },
          {
            "name": "offset",
            "in": "query",
            "type": "integer",
            "required": false,
            "description": "number of pages to skip from the response"
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Contacts returned",
            "schema": {
              "type": "object",
              "properties": {
                "total": {
                  "type": "integer",
                  "format": "int32",
                  "description": "number of contacts found for current user for the given parameters"
                },
                "contacts": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "additional_name": {
                        "type": "string"
                      },
                      "addresses": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "properties": {
                            "address_id": {
                              "type": "string"
                            },
                            "city": {
                              "type": "string"
                            },
                            "country": {
                              "type": "string"
                            },
                            "is_primary": {
                              "type": "boolean"
                            },
                            "label": {
                              "type": "string"
                            },
                            "postal_code": {
                              "type": "string"
                            },
                            "region": {
                              "type": "string"
                            },
                            "street": {
                              "type": "string"
                            },
                            "type": {
                              "type": "string"
                            }
                          },
                          "required": [
                            "address_id",
                            "city"
                          ],
                          "additionalProperties": false
                        }
                      },
                      "avatar": {
                        "type": "string"
                      },
                      "contact_id": {
                        "type": "string"
                      },
                      "date_insert": {
                        "type": "string",
                        "format": "date-time"
                      },
                      "date_update": {
                        "type": "string",
                        "format": "date-time"
                      },
                      "deleted": {
                        "type": "boolean"
                      },
                      "emails": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "properties": {
                            "email_id": {
                              "type": "string"
                            },
                            "address": {
                              "type": "string"
                            },
                            "is_primary": {
                              "type": "boolean"
                            },
                            "label": {
                              "type": "string"
                            },
                            "type": {
                              "type": "string"
                            }
                          },
                          "required": [
                            "email_id",
                            "address"
                          ],
                          "additionalProperties": false
                        }
                      },
                      "family_name": {
                        "type": "string"
                      },
                      "given_name": {
                        "type": "string"
                      },
                      "groups": {
                        "type": "array",
                        "items": {
                          "type": "string"
                        }
                      },
                      "identities": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "allOf": [
                            {
                              "type": "object",
                              "properties": {
                                "infos": {
                                  "type": "object"
                                },
                                "name": {
                                  "type": "string"
                                },
                                "type": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "name"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "identity_id": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "identity_id"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      },
                      "ims": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "allOf": [
                            {
                              "type": "object",
                              "properties": {
                                "address": {
                                  "type": "string"
                                },
                                "is_primary": {
                                  "type": "boolean"
                                },
                                "label": {
                                  "type": "string"
                                },
                                "protocol": {
                                  "type": "string"
                                },
                                "type": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "address"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "im_id": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "im_id"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      },
                      "infos": {
                        "type": "object"
                      },
                      "name_prefix": {
                        "type": "string"
                      },
                      "name_suffix": {
                        "type": "string"
                      },
                      "organizations": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "allOf": [
                            {
                              "type": "object",
                              "properties": {
                                "department": {
                                  "type": "string"
                                },
                                "is_primary": {
                                  "type": "boolean"
                                },
                                "job_description": {
                                  "type": "string"
                                },
                                "label": {
                                  "type": "string"
                                },
                                "name": {
                                  "type": "string"
                                },
                                "title": {
                                  "type": "string"
                                },
                                "type": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "label",
                                "name"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "deleted": {
                                  "type": "boolean"
                                }
                              },
                              "required": [
                                "organization_id"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      },
                      "phones": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "allOf": [
                            {
                              "type": "object",
                              "properties": {
                                "is_primary": {
                                  "type": "boolean"
                                },
                                "number": {
                                  "type": "string"
                                },
                                "type": {
                                  "type": "string"
                                },
                                "uri": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "number"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "phone_id": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "phone_id"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      },
                      "privacy_features": {
                        "type": "object"
                      },
                      "privacy_index": {
                        "type": "integer",
                        "format": "int32"
                      },
                      "public_keys": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "allOf": [
                            {
                              "type": "object",
                              "properties": {
                                "expire_date": {
                                  "type": "string",
                                  "format": "date-time"
                                },
                                "fingerprint": {
                                  "type": "string"
                                },
                                "key": {
                                  "type": "string"
                                },
                                "name": {
                                  "type": "string"
                                },
                                "size": {
                                  "type": "integer",
                                  "format": "int32"
                                }
                              },
                              "required": [
                                "key",
                                "name"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "date_insert": {
                                  "type": "string",
                                  "format": "date-time"
                                },
                                "date_update": {
                                  "type": "string",
                                  "format": "date-time"
                                }
                              },
                              "required": [
                                "date_insert"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      },
                      "tags": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "properties": {
                            "type": {
                              "type": "string",
                              "enum": [
                                "user",
                                "system"
                              ]
                            },
                            "name": {
                              "type": "string"
                            },
                            "tag_id": {
                              "type": "string"
                            }
                          },
                          "required": [
                            "name",
                            "type"
                          ],
                          "additionalProperties": false
                        }
                      },
                      "title": {
                        "type": "string"
                      },
                      "user_id": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "contact_id",
                      "user_id"
                    ],
                    "additionalProperties": false
                  }
                }
              }
            }
          }
        }
      },
      "post": {
        "description": "Create a new contact for the logged-in user",
        "tags": [
          "contacts"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "consumes": [
          "application/json"
        ],
        "parameters": [
          {
            "name": "contact",
            "in": "body",
            "description": "the contact to create",
            "schema": {
              "type": "object",
              "properties": {
                "additional_name": {
                  "type": "string"
                },
                "emails": {
                  "type": "array",
                  "items": {
                    "type": "object"
                  }
                },
                "family_name": {
                  "type": "string"
                },
                "given_name": {
                  "type": "string"
                },
                "groups": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                },
                "ims": {
                  "type": "array",
                  "items": {
                    "type": "object"
                  }
                },
                "infos": {
                  "type": "object"
                },
                "name_prefix": {
                  "type": "string"
                },
                "name_suffix": {
                  "type": "string"
                },
                "organizations": {
                  "type": "array",
                  "items": {
                    "type": "object"
                  }
                },
                "tags": {
                  "type": "array",
                  "items": {
                    "type": "object"
                  }
                }
              }
            }
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Contact created",
            "schema": {
              "type": "object",
              "properties": {
                "location": {
                  "type": "string",
                  "description": "url to retrieve new contact's infos at /contacts/{contact_id}"
                }
              }
            }
          }
        }
      }
    },
    "/v1/contacts/{contact_id}": {
      "get": {
        "description": "Returns a contact",
        "tags": [
          "contacts"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "contact_id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Contact found",
            "schema": {
              "type": "object",
              "properties": {
                "additional_name": {
                  "type": "string"
                },
                "addresses": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "address_id": {
                        "type": "string"
                      },
                      "city": {
                        "type": "string"
                      },
                      "country": {
                        "type": "string"
                      },
                      "is_primary": {
                        "type": "boolean"
                      },
                      "label": {
                        "type": "string"
                      },
                      "postal_code": {
                        "type": "string"
                      },
                      "region": {
                        "type": "string"
                      },
                      "street": {
                        "type": "string"
                      },
                      "type": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "address_id",
                      "city"
                    ],
                    "additionalProperties": false
                  }
                },
                "avatar": {
                  "type": "string"
                },
                "contact_id": {
                  "type": "string"
                },
                "date_insert": {
                  "type": "string",
                  "format": "date-time"
                },
                "date_update": {
                  "type": "string",
                  "format": "date-time"
                },
                "deleted": {
                  "type": "boolean"
                },
                "emails": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "email_id": {
                        "type": "string"
                      },
                      "address": {
                        "type": "string"
                      },
                      "is_primary": {
                        "type": "boolean"
                      },
                      "label": {
                        "type": "string"
                      },
                      "type": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "email_id",
                      "address"
                    ],
                    "additionalProperties": false
                  }
                },
                "family_name": {
                  "type": "string"
                },
                "given_name": {
                  "type": "string"
                },
                "groups": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                },
                "identities": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "allOf": [
                      {
                        "type": "object",
                        "properties": {
                          "infos": {
                            "type": "object"
                          },
                          "name": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "name"
                        ],
                        "additionalProperties": false
                      },
                      {
                        "type": "object",
                        "properties": {
                          "identity_id": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "identity_id"
                        ],
                        "additionalProperties": false
                      }
                    ]
                  }
                },
                "ims": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "allOf": [
                      {
                        "type": "object",
                        "properties": {
                          "address": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "label": {
                            "type": "string"
                          },
                          "protocol": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "address"
                        ],
                        "additionalProperties": false
                      },
                      {
                        "type": "object",
                        "properties": {
                          "im_id": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "im_id"
                        ],
                        "additionalProperties": false
                      }
                    ]
                  }
                },
                "infos": {
                  "type": "object"
                },
                "name_prefix": {
                  "type": "string"
                },
                "name_suffix": {
                  "type": "string"
                },
                "organizations": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "allOf": [
                      {
                        "type": "object",
                        "properties": {
                          "department": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "job_description": {
                            "type": "string"
                          },
                          "label": {
                            "type": "string"
                          },
                          "name": {
                            "type": "string"
                          },
                          "title": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "label",
                          "name"
                        ],
                        "additionalProperties": false
                      },
                      {
                        "type": "object",
                        "properties": {
                          "deleted": {
                            "type": "boolean"
                          }
                        },
                        "required": [
                          "organization_id"
                        ],
                        "additionalProperties": false
                      }
                    ]
                  }
                },
                "phones": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "allOf": [
                      {
                        "type": "object",
                        "properties": {
                          "is_primary": {
                            "type": "boolean"
                          },
                          "number": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          },
                          "uri": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "number"
                        ],
                        "additionalProperties": false
                      },
                      {
                        "type": "object",
                        "properties": {
                          "phone_id": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "phone_id"
                        ],
                        "additionalProperties": false
                      }
                    ]
                  }
                },
                "privacy_features": {
                  "type": "object"
                },
                "privacy_index": {
                  "type": "integer",
                  "format": "int32"
                },
                "public_keys": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "allOf": [
                      {
                        "type": "object",
                        "properties": {
                          "expire_date": {
                            "type": "string",
                            "format": "date-time"
                          },
                          "fingerprint": {
                            "type": "string"
                          },
                          "key": {
                            "type": "string"
                          },
                          "name": {
                            "type": "string"
                          },
                          "size": {
                            "type": "integer",
                            "format": "int32"
                          }
                        },
                        "required": [
                          "key",
                          "name"
                        ],
                        "additionalProperties": false
                      },
                      {
                        "type": "object",
                        "properties": {
                          "date_insert": {
                            "type": "string",
                            "format": "date-time"
                          },
                          "date_update": {
                            "type": "string",
                            "format": "date-time"
                          }
                        },
                        "required": [
                          "date_insert"
                        ],
                        "additionalProperties": false
                      }
                    ]
                  }
                },
                "tags": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "type": {
                        "type": "string",
                        "enum": [
                          "user",
                          "system"
                        ]
                      },
                      "name": {
                        "type": "string"
                      },
                      "tag_id": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "name",
                      "type"
                    ],
                    "additionalProperties": false
                  }
                },
                "title": {
                  "type": "string"
                },
                "user_id": {
                  "type": "string"
                }
              },
              "required": [
                "contact_id",
                "user_id"
              ],
              "additionalProperties": false
            }
          },
          "401": {
            "description": "Unauthorized access",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          },
          "404": {
            "description": "Contact not found",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          }
        }
      },
      "delete": {
        "description": "Not Yet Implemented",
        "tags": [
          "contacts"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "contact_id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "responses": {
          "default": {
            "description": "route not implemented, should raise an error"
          }
        }
      },
      "put": {
        "description": "Not Implemented",
        "tags": [
          "contacts"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "contact_id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "responses": {
          "default": {
            "description": "verb not implemented, should raise an error"
          }
        }
      },
      "patch": {
        "description": "update a contact with rfc5789 and rfc7396 specifications",
        "tags": [
          "contacts"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "contact_id",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "patch",
            "in": "body",
            "required": true,
            "description": "the patch to apply. See 'Caliopen Patch RFC' within /doc directory.",
            "schema": {
              "type": "object",
              "properties": {
                "current_state": {
                  "type": "object",
                  "properties": {
                    "additional_name": {
                      "type": "string"
                    },
                    "addresses": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "address_id": {
                            "type": "string"
                          },
                          "city": {
                            "type": "string"
                          },
                          "country": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "label": {
                            "type": "string"
                          },
                          "postal_code": {
                            "type": "string"
                          },
                          "region": {
                            "type": "string"
                          },
                          "street": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "address_id",
                          "city"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "avatar": {
                      "type": "string"
                    },
                    "contact_id": {
                      "type": "string"
                    },
                    "date_insert": {
                      "type": "string",
                      "format": "date-time"
                    },
                    "date_update": {
                      "type": "string",
                      "format": "date-time"
                    },
                    "deleted": {
                      "type": "boolean"
                    },
                    "emails": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "email_id": {
                            "type": "string"
                          },
                          "address": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "label": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "email_id",
                          "address"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "family_name": {
                      "type": "string"
                    },
                    "given_name": {
                      "type": "string"
                    },
                    "groups": {
                      "type": "array",
                      "items": {
                        "type": "string"
                      }
                    },
                    "identities": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "infos": {
                                "type": "object"
                              },
                              "name": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "identity_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "identity_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "ims": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "address": {
                                "type": "string"
                              },
                              "is_primary": {
                                "type": "boolean"
                              },
                              "label": {
                                "type": "string"
                              },
                              "protocol": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "address"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "im_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "im_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "infos": {
                      "type": "object"
                    },
                    "name_prefix": {
                      "type": "string"
                    },
                    "name_suffix": {
                      "type": "string"
                    },
                    "organizations": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "department": {
                                "type": "string"
                              },
                              "is_primary": {
                                "type": "boolean"
                              },
                              "job_description": {
                                "type": "string"
                              },
                              "label": {
                                "type": "string"
                              },
                              "name": {
                                "type": "string"
                              },
                              "title": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "label",
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "deleted": {
                                "type": "boolean"
                              }
                            },
                            "required": [
                              "organization_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "phones": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "is_primary": {
                                "type": "boolean"
                              },
                              "number": {
                                "type": "string"
                              },
                              "type": {
                                "type": "string"
                              },
                              "uri": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "number"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "phone_id": {
                                "type": "string"
                              }
                            },
                            "required": [
                              "phone_id"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "privacy_features": {
                      "type": "object"
                    },
                    "privacy_index": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "public_keys": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "allOf": [
                          {
                            "type": "object",
                            "properties": {
                              "expire_date": {
                                "type": "string",
                                "format": "date-time"
                              },
                              "fingerprint": {
                                "type": "string"
                              },
                              "key": {
                                "type": "string"
                              },
                              "name": {
                                "type": "string"
                              },
                              "size": {
                                "type": "integer",
                                "format": "int32"
                              }
                            },
                            "required": [
                              "key",
                              "name"
                            ],
                            "additionalProperties": false
                          },
                          {
                            "type": "object",
                            "properties": {
                              "date_insert": {
                                "type": "string",
                                "format": "date-time"
                              },
                              "date_update": {
                                "type": "string",
                                "format": "date-time"
                              }
                            },
                            "required": [
                              "date_insert"
                            ],
                            "additionalProperties": false
                          }
                        ]
                      }
                    },
                    "tags": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "type": {
                            "type": "string",
                            "enum": [
                              "user",
                              "system"
                            ]
                          },
                          "name": {
                            "type": "string"
                          },
                          "tag_id": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "name",
                          "type"
                        ],
                        "additionalProperties": false
                      }
                    },
                    "title": {
                      "type": "string"
                    },
                    "user_id": {
                      "type": "string"
                    }
                  }
                },
                "additional_name": {
                  "type": "string"
                },
                "addresses": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "address_id": {
                        "type": "string"
                      },
                      "city": {
                        "type": "string"
                      },
                      "country": {
                        "type": "string"
                      },
                      "is_primary": {
                        "type": "boolean"
                      },
                      "label": {
                        "type": "string"
                      },
                      "postal_code": {
                        "type": "string"
                      },
                      "region": {
                        "type": "string"
                      },
                      "street": {
                        "type": "string"
                      },
                      "type": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "address_id",
                      "city"
                    ],
                    "additionalProperties": false
                  }
                },
                "avatar": {
                  "type": "string"
                },
                "contact_id": {
                  "type": "string"
                },
                "date_insert": {
                  "type": "string",
                  "format": "date-time"
                },
                "date_update": {
                  "type": "string",
                  "format": "date-time"
                },
                "deleted": {
                  "type": "boolean"
                },
                "emails": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "email_id": {
                        "type": "string"
                      },
                      "address": {
                        "type": "string"
                      },
                      "is_primary": {
                        "type": "boolean"
                      },
                      "label": {
                        "type": "string"
                      },
                      "type": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "email_id",
                      "address"
                    ],
                    "additionalProperties": false
                  }
                },
                "family_name": {
                  "type": "string"
                },
                "given_name": {
                  "type": "string"
                },
                "groups": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                },
                "identities": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "allOf": [
                      {
                        "type": "object",
                        "properties": {
                          "infos": {
                            "type": "object"
                          },
                          "name": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "name"
                        ],
                        "additionalProperties": false
                      },
                      {
                        "type": "object",
                        "properties": {
                          "identity_id": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "identity_id"
                        ],
                        "additionalProperties": false
                      }
                    ]
                  }
                },
                "ims": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "allOf": [
                      {
                        "type": "object",
                        "properties": {
                          "address": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "label": {
                            "type": "string"
                          },
                          "protocol": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "address"
                        ],
                        "additionalProperties": false
                      },
                      {
                        "type": "object",
                        "properties": {
                          "im_id": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "im_id"
                        ],
                        "additionalProperties": false
                      }
                    ]
                  }
                },
                "infos": {
                  "type": "object"
                },
                "name_prefix": {
                  "type": "string"
                },
                "name_suffix": {
                  "type": "string"
                },
                "organizations": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "allOf": [
                      {
                        "type": "object",
                        "properties": {
                          "department": {
                            "type": "string"
                          },
                          "is_primary": {
                            "type": "boolean"
                          },
                          "job_description": {
                            "type": "string"
                          },
                          "label": {
                            "type": "string"
                          },
                          "name": {
                            "type": "string"
                          },
                          "title": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "label",
                          "name"
                        ],
                        "additionalProperties": false
                      },
                      {
                        "type": "object",
                        "properties": {
                          "deleted": {
                            "type": "boolean"
                          }
                        },
                        "required": [
                          "organization_id"
                        ],
                        "additionalProperties": false
                      }
                    ]
                  }
                },
                "phones": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "allOf": [
                      {
                        "type": "object",
                        "properties": {
                          "is_primary": {
                            "type": "boolean"
                          },
                          "number": {
                            "type": "string"
                          },
                          "type": {
                            "type": "string"
                          },
                          "uri": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "number"
                        ],
                        "additionalProperties": false
                      },
                      {
                        "type": "object",
                        "properties": {
                          "phone_id": {
                            "type": "string"
                          }
                        },
                        "required": [
                          "phone_id"
                        ],
                        "additionalProperties": false
                      }
                    ]
                  }
                },
                "privacy_features": {
                  "type": "object"
                },
                "privacy_index": {
                  "type": "integer",
                  "format": "int32"
                },
                "public_keys": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "allOf": [
                      {
                        "type": "object",
                        "properties": {
                          "expire_date": {
                            "type": "string",
                            "format": "date-time"
                          },
                          "fingerprint": {
                            "type": "string"
                          },
                          "key": {
                            "type": "string"
                          },
                          "name": {
                            "type": "string"
                          },
                          "size": {
                            "type": "integer",
                            "format": "int32"
                          }
                        },
                        "required": [
                          "key",
                          "name"
                        ],
                        "additionalProperties": false
                      },
                      {
                        "type": "object",
                        "properties": {
                          "date_insert": {
                            "type": "string",
                            "format": "date-time"
                          },
                          "date_update": {
                            "type": "string",
                            "format": "date-time"
                          }
                        },
                        "required": [
                          "date_insert"
                        ],
                        "additionalProperties": false
                      }
                    ]
                  }
                },
                "tags": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "type": {
                        "type": "string",
                        "enum": [
                          "user",
                          "system"
                        ]
                      },
                      "name": {
                        "type": "string"
                      },
                      "tag_id": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "name",
                      "type"
                    ],
                    "additionalProperties": false
                  }
                },
                "title": {
                  "type": "string"
                },
                "user_id": {
                  "type": "string"
                }
              },
              "required": [
                "current_state"
              ]
            }
          }
        ],
        "consumes": [
          "application/json"
        ],
        "responses": {
          "204": {
            "description": "Update successful. No body is returned."
          },
          "400": {
            "description": "json payload malformed",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          },
          "404": {
            "description": "contact not found",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          },
          "422": {
            "description": "patch json was malformed or unprocessable",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/v1/discussions": {
      "get": {
        "description": "Returns discussions belonging to current user according to given parameters",
        "tags": [
          "discussions"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "X-Caliopen-PI",
            "in": "header",
            "required": true,
            "description": "The PI range requested in form of 1;100",
            "type": "string",
            "default": "1;100"
          },
          {
            "name": "limit",
            "in": "query",
            "required": false,
            "type": "integer",
            "description": "number of discussions to return per page"
          },
          {
            "name": "offset",
            "in": "query",
            "type": "integer",
            "required": false,
            "description": "number of pages to skip from the response"
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Discussions returned",
            "schema": {
              "type": "object",
              "properties": {
                "total": {
                  "type": "integer",
                  "format": "int32",
                  "description": "number of discussions found for current user for the given parameters"
                },
                "discussion": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "attachment_count": {
                        "type": "integer",
                        "format": "int32"
                      },
                      "contacts": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "properties": {
                            "additional_name": {
                              "type": "string"
                            },
                            "addresses": {
                              "type": "array",
                              "items": {
                                "type": "object",
                                "properties": {
                                  "address_id": {
                                    "type": "string"
                                  },
                                  "city": {
                                    "type": "string"
                                  },
                                  "country": {
                                    "type": "string"
                                  },
                                  "is_primary": {
                                    "type": "boolean"
                                  },
                                  "label": {
                                    "type": "string"
                                  },
                                  "postal_code": {
                                    "type": "string"
                                  },
                                  "region": {
                                    "type": "string"
                                  },
                                  "street": {
                                    "type": "string"
                                  },
                                  "type": {
                                    "type": "string"
                                  }
                                },
                                "required": [
                                  "address_id",
                                  "city"
                                ],
                                "additionalProperties": false
                              }
                            },
                            "avatar": {
                              "type": "string"
                            },
                            "contact_id": {
                              "type": "string"
                            },
                            "date_insert": {
                              "type": "string",
                              "format": "date-time"
                            },
                            "date_update": {
                              "type": "string",
                              "format": "date-time"
                            },
                            "deleted": {
                              "type": "boolean"
                            },
                            "emails": {
                              "type": "array",
                              "items": {
                                "type": "object",
                                "properties": {
                                  "email_id": {
                                    "type": "string"
                                  },
                                  "address": {
                                    "type": "string"
                                  },
                                  "is_primary": {
                                    "type": "boolean"
                                  },
                                  "label": {
                                    "type": "string"
                                  },
                                  "type": {
                                    "type": "string"
                                  }
                                },
                                "required": [
                                  "email_id",
                                  "address"
                                ],
                                "additionalProperties": false
                              }
                            },
                            "family_name": {
                              "type": "string"
                            },
                            "given_name": {
                              "type": "string"
                            },
                            "groups": {
                              "type": "array",
                              "items": {
                                "type": "string"
                              }
                            },
                            "identities": {
                              "type": "array",
                              "items": {
                                "type": "object",
                                "allOf": [
                                  {
                                    "type": "object",
                                    "properties": {
                                      "infos": {
                                        "type": "object"
                                      },
                                      "name": {
                                        "type": "string"
                                      },
                                      "type": {
                                        "type": "string"
                                      }
                                    },
                                    "required": [
                                      "name"
                                    ],
                                    "additionalProperties": false
                                  },
                                  {
                                    "type": "object",
                                    "properties": {
                                      "identity_id": {
                                        "type": "string"
                                      }
                                    },
                                    "required": [
                                      "identity_id"
                                    ],
                                    "additionalProperties": false
                                  }
                                ]
                              }
                            },
                            "ims": {
                              "type": "array",
                              "items": {
                                "type": "object",
                                "allOf": [
                                  {
                                    "type": "object",
                                    "properties": {
                                      "address": {
                                        "type": "string"
                                      },
                                      "is_primary": {
                                        "type": "boolean"
                                      },
                                      "label": {
                                        "type": "string"
                                      },
                                      "protocol": {
                                        "type": "string"
                                      },
                                      "type": {
                                        "type": "string"
                                      }
                                    },
                                    "required": [
                                      "address"
                                    ],
                                    "additionalProperties": false
                                  },
                                  {
                                    "type": "object",
                                    "properties": {
                                      "im_id": {
                                        "type": "string"
                                      }
                                    },
                                    "required": [
                                      "im_id"
                                    ],
                                    "additionalProperties": false
                                  }
                                ]
                              }
                            },
                            "infos": {
                              "type": "object"
                            },
                            "name_prefix": {
                              "type": "string"
                            },
                            "name_suffix": {
                              "type": "string"
                            },
                            "organizations": {
                              "type": "array",
                              "items": {
                                "type": "object",
                                "allOf": [
                                  {
                                    "type": "object",
                                    "properties": {
                                      "department": {
                                        "type": "string"
                                      },
                                      "is_primary": {
                                        "type": "boolean"
                                      },
                                      "job_description": {
                                        "type": "string"
                                      },
                                      "label": {
                                        "type": "string"
                                      },
                                      "name": {
                                        "type": "string"
                                      },
                                      "title": {
                                        "type": "string"
                                      },
                                      "type": {
                                        "type": "string"
                                      }
                                    },
                                    "required": [
                                      "label",
                                      "name"
                                    ],
                                    "additionalProperties": false
                                  },
                                  {
                                    "type": "object",
                                    "properties": {
                                      "deleted": {
                                        "type": "boolean"
                                      }
                                    },
                                    "required": [
                                      "organization_id"
                                    ],
                                    "additionalProperties": false
                                  }
                                ]
                              }
                            },
                            "phones": {
                              "type": "array",
                              "items": {
                                "type": "object",
                                "allOf": [
                                  {
                                    "type": "object",
                                    "properties": {
                                      "is_primary": {
                                        "type": "boolean"
                                      },
                                      "number": {
                                        "type": "string"
                                      },
                                      "type": {
                                        "type": "string"
                                      },
                                      "uri": {
                                        "type": "string"
                                      }
                                    },
                                    "required": [
                                      "number"
                                    ],
                                    "additionalProperties": false
                                  },
                                  {
                                    "type": "object",
                                    "properties": {
                                      "phone_id": {
                                        "type": "string"
                                      }
                                    },
                                    "required": [
                                      "phone_id"
                                    ],
                                    "additionalProperties": false
                                  }
                                ]
                              }
                            },
                            "privacy_features": {
                              "type": "object"
                            },
                            "privacy_index": {
                              "type": "integer",
                              "format": "int32"
                            },
                            "public_keys": {
                              "type": "array",
                              "items": {
                                "type": "object",
                                "allOf": [
                                  {
                                    "type": "object",
                                    "properties": {
                                      "expire_date": {
                                        "type": "string",
                                        "format": "date-time"
                                      },
                                      "fingerprint": {
                                        "type": "string"
                                      },
                                      "key": {
                                        "type": "string"
                                      },
                                      "name": {
                                        "type": "string"
                                      },
                                      "size": {
                                        "type": "integer",
                                        "format": "int32"
                                      }
                                    },
                                    "required": [
                                      "key",
                                      "name"
                                    ],
                                    "additionalProperties": false
                                  },
                                  {
                                    "type": "object",
                                    "properties": {
                                      "date_insert": {
                                        "type": "string",
                                        "format": "date-time"
                                      },
                                      "date_update": {
                                        "type": "string",
                                        "format": "date-time"
                                      }
                                    },
                                    "required": [
                                      "date_insert"
                                    ],
                                    "additionalProperties": false
                                  }
                                ]
                              }
                            },
                            "tags": {
                              "type": "array",
                              "items": {
                                "type": "object",
                                "properties": {
                                  "type": {
                                    "type": "string",
                                    "enum": [
                                      "user",
                                      "system"
                                    ]
                                  },
                                  "name": {
                                    "type": "string"
                                  },
                                  "tag_id": {
                                    "type": "string"
                                  }
                                },
                                "required": [
                                  "name",
                                  "type"
                                ],
                                "additionalProperties": false
                              }
                            },
                            "title": {
                              "type": "string"
                            },
                            "user_id": {
                              "type": "string"
                            }
                          },
                          "required": [
                            "contact_id",
                            "user_id"
                          ],
                          "additionalProperties": false
                        }
                      },
                      "date_insert": {
                        "type": "string",
                        "format": "date-time"
                      },
                      "date_update": {
                        "type": "string",
                        "format": "date-time"
                      },
                      "importance_level": {
                        "type": "integer",
                        "format": "int32"
                      },
                      "privacy_index": {
                        "type": "integer",
                        "format": "int32"
                      },
                      "tags": {
                        "type": "array",
                        "items": {
                          "type": "string"
                        }
                      },
                      "text": {
                        "type": "string"
                      },
                      "discussion_id": {
                        "type": "string"
                      },
                      "total_count": {
                        "type": "integer",
                        "format": "int32"
                      },
                      "unread_count": {
                        "type": "integer",
                        "format": "int32"
                      }
                    }
                  }
                }
              }
            }
          }
        }
      },
      "post": {
        "description": "Not Yet Implemented. Should start a new discussion",
        "tags": [
          "discussions"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "responses": {
          "default": {
            "description": "route not implemented, should raise an error"
          }
        }
      }
    },
    "/v1/discussions/{discussion_id}": {
      "get": {
        "description": "Returns a discussion",
        "tags": [
          "discussions"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "discussion_id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Discussion found",
            "schema": {
              "type": "object",
              "properties": {
                "attachment_count": {
                  "type": "integer",
                  "format": "int32"
                },
                "contacts": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "additional_name": {
                        "type": "string"
                      },
                      "addresses": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "properties": {
                            "address_id": {
                              "type": "string"
                            },
                            "city": {
                              "type": "string"
                            },
                            "country": {
                              "type": "string"
                            },
                            "is_primary": {
                              "type": "boolean"
                            },
                            "label": {
                              "type": "string"
                            },
                            "postal_code": {
                              "type": "string"
                            },
                            "region": {
                              "type": "string"
                            },
                            "street": {
                              "type": "string"
                            },
                            "type": {
                              "type": "string"
                            }
                          },
                          "required": [
                            "address_id",
                            "city"
                          ],
                          "additionalProperties": false
                        }
                      },
                      "avatar": {
                        "type": "string"
                      },
                      "contact_id": {
                        "type": "string"
                      },
                      "date_insert": {
                        "type": "string",
                        "format": "date-time"
                      },
                      "date_update": {
                        "type": "string",
                        "format": "date-time"
                      },
                      "deleted": {
                        "type": "boolean"
                      },
                      "emails": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "properties": {
                            "email_id": {
                              "type": "string"
                            },
                            "address": {
                              "type": "string"
                            },
                            "is_primary": {
                              "type": "boolean"
                            },
                            "label": {
                              "type": "string"
                            },
                            "type": {
                              "type": "string"
                            }
                          },
                          "required": [
                            "email_id",
                            "address"
                          ],
                          "additionalProperties": false
                        }
                      },
                      "family_name": {
                        "type": "string"
                      },
                      "given_name": {
                        "type": "string"
                      },
                      "groups": {
                        "type": "array",
                        "items": {
                          "type": "string"
                        }
                      },
                      "identities": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "allOf": [
                            {
                              "type": "object",
                              "properties": {
                                "infos": {
                                  "type": "object"
                                },
                                "name": {
                                  "type": "string"
                                },
                                "type": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "name"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "identity_id": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "identity_id"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      },
                      "ims": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "allOf": [
                            {
                              "type": "object",
                              "properties": {
                                "address": {
                                  "type": "string"
                                },
                                "is_primary": {
                                  "type": "boolean"
                                },
                                "label": {
                                  "type": "string"
                                },
                                "protocol": {
                                  "type": "string"
                                },
                                "type": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "address"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "im_id": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "im_id"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      },
                      "infos": {
                        "type": "object"
                      },
                      "name_prefix": {
                        "type": "string"
                      },
                      "name_suffix": {
                        "type": "string"
                      },
                      "organizations": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "allOf": [
                            {
                              "type": "object",
                              "properties": {
                                "department": {
                                  "type": "string"
                                },
                                "is_primary": {
                                  "type": "boolean"
                                },
                                "job_description": {
                                  "type": "string"
                                },
                                "label": {
                                  "type": "string"
                                },
                                "name": {
                                  "type": "string"
                                },
                                "title": {
                                  "type": "string"
                                },
                                "type": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "label",
                                "name"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "deleted": {
                                  "type": "boolean"
                                }
                              },
                              "required": [
                                "organization_id"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      },
                      "phones": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "allOf": [
                            {
                              "type": "object",
                              "properties": {
                                "is_primary": {
                                  "type": "boolean"
                                },
                                "number": {
                                  "type": "string"
                                },
                                "type": {
                                  "type": "string"
                                },
                                "uri": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "number"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "phone_id": {
                                  "type": "string"
                                }
                              },
                              "required": [
                                "phone_id"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      },
                      "privacy_features": {
                        "type": "object"
                      },
                      "privacy_index": {
                        "type": "integer",
                        "format": "int32"
                      },
                      "public_keys": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "allOf": [
                            {
                              "type": "object",
                              "properties": {
                                "expire_date": {
                                  "type": "string",
                                  "format": "date-time"
                                },
                                "fingerprint": {
                                  "type": "string"
                                },
                                "key": {
                                  "type": "string"
                                },
                                "name": {
                                  "type": "string"
                                },
                                "size": {
                                  "type": "integer",
                                  "format": "int32"
                                }
                              },
                              "required": [
                                "key",
                                "name"
                              ],
                              "additionalProperties": false
                            },
                            {
                              "type": "object",
                              "properties": {
                                "date_insert": {
                                  "type": "string",
                                  "format": "date-time"
                                },
                                "date_update": {
                                  "type": "string",
                                  "format": "date-time"
                                }
                              },
                              "required": [
                                "date_insert"
                              ],
                              "additionalProperties": false
                            }
                          ]
                        }
                      },
                      "tags": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "properties": {
                            "type": {
                              "type": "string",
                              "enum": [
                                "user",
                                "system"
                              ]
                            },
                            "name": {
                              "type": "string"
                            },
                            "tag_id": {
                              "type": "string"
                            }
                          },
                          "required": [
                            "name",
                            "type"
                          ],
                          "additionalProperties": false
                        }
                      },
                      "title": {
                        "type": "string"
                      },
                      "user_id": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "contact_id",
                      "user_id"
                    ],
                    "additionalProperties": false
                  }
                },
                "date_insert": {
                  "type": "string",
                  "format": "date-time"
                },
                "date_update": {
                  "type": "string",
                  "format": "date-time"
                },
                "importance_level": {
                  "type": "integer",
                  "format": "int32"
                },
                "privacy_index": {
                  "type": "integer",
                  "format": "int32"
                },
                "tags": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                },
                "text": {
                  "type": "string"
                },
                "discussion_id": {
                  "type": "string"
                },
                "total_count": {
                  "type": "integer",
                  "format": "int32"
                },
                "unread_count": {
                  "type": "integer",
                  "format": "int32"
                }
              }
            }
          },
          "401": {
            "description": "Unauthorized access",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          },
          "404": {
            "description": "Discussion not found",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/v1/discussions/{discussion_id}/messages": {
      "get": {
        "description": "Returns messages belonging to a discussion according to given parameters",
        "tags": [
          "discussions",
          "messages"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "discussion_id",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "X-Caliopen-PI",
            "in": "header",
            "required": true,
            "description": "The PI range requested in form of 1;100",
            "type": "string",
            "default": "1;100"
          },
          {
            "name": "limit",
            "in": "query",
            "required": false,
            "type": "integer",
            "description": "number of discussions to return per page"
          },
          {
            "name": "offset",
            "in": "query",
            "type": "integer",
            "required": false,
            "description": "number of pages to skip from the response"
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "messages returned",
            "schema": {
              "type": "object",
              "properties": {
                "total": {
                  "type": "integer",
                  "format": "int32",
                  "description": "number of messages found for the discussion for the given parameters"
                },
                "messages": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "date": {
                        "type": "string",
                        "format": "date-time"
                      },
                      "date_insert": {
                        "type": "string",
                        "format": "date-time"
                      },
                      "external_message_id": {
                        "type": "string"
                      },
                      "external_parent_id": {
                        "type": "string"
                      },
                      "external_thread_id": {
                        "type": "string"
                      },
                      "from_": {
                        "type": "string"
                      },
                      "headers": {
                        "type": "object"
                      },
                      "importance_level": {
                        "type": "integer",
                        "format": "int32"
                      },
                      "message_id": {
                        "type": "string"
                      },
                      "parts": {
                        "type": "array",
                        "items": {
                          "type": "string"
                        }
                      },
                      "privacy_index": {
                        "type": "integer",
                        "format": "int32"
                      },
                      "recipients": {
                        "type": "array",
                        "items": {
                          "type": "object",
                          "properties": {
                            "address": {
                              "type": "string"
                            },
                            "contact_id": {
                              "type": "string"
                            },
                            "label": {
                              "type": "string"
                            },
                            "protocol": {
                              "type": "string"
                            },
                            "type": {
                              "type": "string"
                            }
                          },
                          "required": [
                            "address",
                            "type"
                          ],
                          "additionalProperties": false
                        }
                      },
                      "size": {
                        "type": "integer",
                        "format": "int32"
                      },
                      "state": {
                        "type": "string"
                      },
                      "subject": {
                        "type": "string"
                      },
                      "tags": {
                        "type": "array",
                        "items": {
                          "type": "string"
                        }
                      },
                      "text": {
                        "type": "string"
                      },
                      "type": {
                        "type": "string"
                      }
                    }
                  }
                }
              }
            }
          },
          "401": {
            "description": "Unauthorized access",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          },
          "404": {
            "description": "Discussion not found",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          }
        }
      },
      "post": {
        "description": "##TO BE COMPLETED - not working## Add a new message to the discussion",
        "tags": [
          "discussions",
          "messages"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "consumes": [
          "application/json"
        ],
        "parameters": [
          {
            "name": "discussion_id",
            "in": "path",
            "required": true,
            "type": "string"
          },
          {
            "name": "message",
            "in": "body",
            "description": "The new message",
            "schema": {
              "type": "object",
              "properties": {
                "bcc_recipients": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "address": {
                        "type": "string"
                      },
                      "contact_id": {
                        "type": "string"
                      },
                      "label": {
                        "type": "string"
                      },
                      "protocol": {
                        "type": "string"
                      },
                      "type": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "address",
                      "type"
                    ],
                    "additionalProperties": false
                  }
                },
                "cc_recipients": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "address": {
                        "type": "string"
                      },
                      "contact_id": {
                        "type": "string"
                      },
                      "label": {
                        "type": "string"
                      },
                      "protocol": {
                        "type": "string"
                      },
                      "type": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "address",
                      "type"
                    ],
                    "additionalProperties": false
                  }
                },
                "reply_to": {
                  "type": "string"
                },
                "subject": {
                  "type": "string"
                },
                "tags": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                },
                "text": {
                  "type": "string"
                },
                "to_recipients": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "address": {
                        "type": "string"
                      },
                      "contact_id": {
                        "type": "string"
                      },
                      "label": {
                        "type": "string"
                      },
                      "protocol": {
                        "type": "string"
                      },
                      "type": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "address",
                      "type"
                    ],
                    "additionalProperties": false
                  }
                }
              }
            }
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Message created",
            "schema": {
              "type": "object",
              "properties": {
                "date": {
                  "type": "string",
                  "format": "date-time"
                },
                "date_insert": {
                  "type": "string",
                  "format": "date-time"
                },
                "external_message_id": {
                  "type": "string"
                },
                "external_parent_id": {
                  "type": "string"
                },
                "external_thread_id": {
                  "type": "string"
                },
                "from_": {
                  "type": "string"
                },
                "headers": {
                  "type": "object"
                },
                "importance_level": {
                  "type": "integer",
                  "format": "int32"
                },
                "message_id": {
                  "type": "string"
                },
                "parts": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                },
                "privacy_index": {
                  "type": "integer",
                  "format": "int32"
                },
                "recipients": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "address": {
                        "type": "string"
                      },
                      "contact_id": {
                        "type": "string"
                      },
                      "label": {
                        "type": "string"
                      },
                      "protocol": {
                        "type": "string"
                      },
                      "type": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "address",
                      "type"
                    ],
                    "additionalProperties": false
                  }
                },
                "size": {
                  "type": "integer",
                  "format": "int32"
                },
                "state": {
                  "type": "string"
                },
                "subject": {
                  "type": "string"
                },
                "tags": {
                  "type": "array",
                  "items": {
                    "type": "string"
                  }
                },
                "text": {
                  "type": "string"
                },
                "type": {
                  "type": "string"
                }
              }
            }
          }
        }
      }
    },
    "/v1/raws/{raw_msg_id}": {
      "get": {
        "description": "Returns a raw message",
        "tags": [
          "messages"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "raw_msg_id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "produces": [
          "text/plain"
        ],
        "responses": {
          "200": {
            "description": "the raw message",
            "schema": {
              "type": "string"
            }
          }
        }
      }
    },
    "/v1/tags": {
      "get": {
        "description": "Returns tags visible to current user according to given parameters",
        "tags": [
          "tags"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Tags returned",
            "schema": {
              "type": "object",
              "properties": {
                "total": {
                  "type": "integer",
                  "format": "int32",
                  "description": "number of tags found for user for the given parameters"
                },
                "tags": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "type": {
                        "type": "string",
                        "enum": [
                          "user",
                          "system"
                        ]
                      },
                      "name": {
                        "type": "string"
                      },
                      "tag_id": {
                        "type": "string"
                      }
                    },
                    "required": [
                      "name",
                      "type"
                    ],
                    "additionalProperties": false
                  }
                }
              }
            }
          }
        }
      },
      "post": {
        "description": "Create a new Tag for an user",
        "tags": [
          "tags"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "consumes": [
          "application/json"
        ],
        "parameters": [
          {
            "name": "tag",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "properties": {
                "name": {
                  "type": "string"
                }
              },
              "required": [
                "name"
              ],
              "additionalProperties": false
            }
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "User tag creation completed",
            "schema": {
              "type": "object",
              "properties": {
                "location": {
                  "type": "string",
                  "description": "url to retrieve new tag's infos at /tags/{name}"
                }
              }
            }
          }
        }
      }
    },
    "/v1/tags/{tag_id}": {
      "get": {
        "description": "Retrieve tag infos",
        "tags": [
          "tags"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "tag_id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "Successful response with json object",
            "schema": {
              "type": "object",
              "properties": {
                "type": {
                  "type": "string",
                  "enum": [
                    "user",
                    "system"
                  ]
                },
                "name": {
                  "type": "string"
                },
                "tag_id": {
                  "type": "string"
                }
              },
              "required": [
                "name",
                "type"
              ],
              "additionalProperties": false
            }
          },
          "401": {
            "description": "Unauthorized access",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          }
        }
      },
      "delete": {
        "description": "Delete a tag belonging to an user",
        "tags": [
          "tags"
        ],
        "security": [
          {
            "basicAuth": []
          }
        ],
        "parameters": [
          {
            "name": "tag_id",
            "in": "path",
            "required": true,
            "type": "string"
          }
        ],
        "responses": {
          "200": {
            "description": "Successful deletion"
          },
          "401": {
            "description": "Unauthorized access",
            "schema": {
              "type": "object",
              "properties": {
                "error": {
                  "type": "object",
                  "properties": {
                    "message": {
                      "type": "string"
                    },
                    "code": {
                      "type": "integer",
                      "format": "int32"
                    },
                    "name": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  "securityDefinitions": {
    "basicAuth": {
      "type": "basic",
      "description": "HTTP Basic Authentication. Password is the access_token return by /authentications and Username is the user_id returned by /authentications"
    }
  }
}
`