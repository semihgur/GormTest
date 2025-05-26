package main

import (
	"GormMany2ManyTest/domain"
	"GormMany2ManyTest/gormfilter"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/schema"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"net/http"
	"reflect"
	"strings"
)

var db *gorm.DB
var err error
var decoder = schema.NewDecoder()

func main() {

	// Connect to the database
	db, err = connectDB()
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&domain.User{}, &domain.Devices{}, &domain.Groups{}, &domain.Permission{})
	if err != nil {
		fmt.Println("Failed to migrate database:", err)
		return
	}

	//createData()

	// Set up chi router
	r := chi.NewRouter()

	// Define the endpoint
	r.Get("/user/{id}", userInfoGetHandlerV2)
	r.Get("/userv2", GetUsersHandler)

	// Start the server
	http.ListenAndServe(":8080", r)
}

func connectDB() (*gorm.DB, error) {
	// Database connection string
	dsn := "host=localhost user=postgres password=tayitkan dbname=gormtest port=5432 sslmode=disable TimeZone=Asia/Istanbul"

	var err error
	// Connect to PostgreSQL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func userInfoGetHandlerV2(w http.ResponseWriter, r *http.Request) {
	// Start building the GORM query on the User model
	query := db.Model(&domain.User{})

	// Reflect type of User for association matching
	userType := reflect.TypeOf(domain.User{})

	mapFilters(query, r, userType)

	// Execute query: preload conditions applied
	var users []domain.User
	if err := query.Find(&users).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func createData() {

	// Create some sample data
	permission1 := domain.Permission{Code: "read"}
	permission2 := domain.Permission{Code: "write"}
	permission3 := domain.Permission{Code: "execute"}
	group1 := domain.Groups{Name: "admin", Permissions: []domain.Permission{permission3, permission2}}
	group2 := domain.Groups{Name: "user", Permissions: []domain.Permission{permission1}}
	device1 := domain.Devices{Name: "device1"}
	device2 := domain.Devices{Name: "device2"}
	user := domain.User{Name: "John Doe", Devices: []domain.Devices{device1, device2}, Groups: []domain.Groups{group1, group2}}

	// Save to database
	if err = db.Create(&user).Error; err != nil {
		fmt.Println("Failed to create data:", err)
		return
	}

	fmt.Println("Data created successfully")
}

func mapFilters(query *gorm.DB, r *http.Request, currentType reflect.Type) {
	// Data structure to hold association filters:
	// map[assocPath] = map[column] = []values
	filters := make(map[string]map[string][]interface{})

	// Helper: capitalize and pluralize association names as needed.
	// For example, "role" → "Roles", "group_permission" → "Groups.Permissions".
	capitalize := func(s string) string {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	// For simplicity, assume add "s" for plural; adjust for irregulars in real code.
	pluralize := func(name string) string {
		// naive pluralization: add 's' (or use a library for complex cases)
		if strings.HasSuffix(name, "s") {
			return name
		}
		return name + "s"
	}

	// Parse form values
	err := r.ParseForm()
	if err != nil {
		return
	}

	// Build filters map from form values
	for key, values := range r.Form {
		parts := strings.Split(key, "_")
		if len(parts) < 2 {
			// No underscore: could be main User field. (Not covered here.)
			continue
		}
		// Last part is column, preceding parts form the association path
		colName := parts[len(parts)-1]
		assocParts := parts[:len(parts)-1]

		// Build association path string (e.g. "Roles.Permissions")
		// by matching parts to struct field names
		fullPath := ""
		for _, part := range assocParts {
			// Capitalize the part (e.g. "role" -> "Role")
			partName := capitalize(part)
			// Try to find a matching field in currentType
			found := ""
			for j := 0; j < currentType.NumField(); j++ {
				field := currentType.Field(j)
				// Match lower-case field name with partName or its plural
				if strings.EqualFold(field.Name, partName) ||
					strings.EqualFold(field.Name, pluralize(partName)) {
					found = field.Name
					// If this is a slice (many), update currentType to element type
					if field.Type.Kind() == reflect.Slice {
						currentType = field.Type.Elem()
					} else {
						currentType = field.Type
					}
					break
				}
			}
			if found == "" {
				// No matching field: skip this param
				fullPath = ""
				break
			}
			// Append to fullPath
			if fullPath == "" {
				fullPath = found
			} else {
				fullPath = fullPath + "." + found
			}
		}
		if fullPath == "" {
			// could not map association; skip
			continue
		}
		// Initialize map for this association if needed
		if _, ok := filters[fullPath]; !ok {
			filters[fullPath] = make(map[string][]interface{})
		}
		// Convert values to []interface{} for GORM
		ifaceVals := make([]interface{}, len(values))
		for i, v := range values {
			ifaceVals[i] = v
		}
		// Store under column name (assume DB column = struct field in snake_case)
		filters[fullPath][colName] = append(filters[fullPath][colName], ifaceVals...)
	}

	// Apply Preload for each association path
	for assocPath, colMap := range filters {
		// If no specific filters for this assoc, just preload all:
		if len(colMap) == 0 {
			query = query.Preload(assocPath)
			continue
		}
		// Preload with conditions in a closure
		query = query.Preload(assocPath, func(db *gorm.DB) *gorm.DB {
			for col, vals := range colMap {
				// Build condition, using IN for multiple values
				placeholder := strings.Repeat("?,", len(vals))
				placeholder = placeholder[:len(placeholder)-1] // remove trailing comma
				// Example: "roles.id IN (?)"
				cond := fmt.Sprintf("%s IN (?)", col)
				db = db.Where(cond, vals)
			}
			return db
		})
	}
}

func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	// 1) Parse the query string (e.g. ?user_name=Alice&role_id=2&permission=READ)
	if err = r.ParseForm(); err != nil {
		http.Error(w, "Invalid query", http.StatusBadRequest)
		return
	}

	// 2) Decode into UserFilter
	var filter domain.UserFilterV3
	if err := decoder.Decode(&filter, r.Form); err != nil {
		http.Error(w, "Bad query parameters", http.StatusBadRequest)
		return
	}

	query := db.Model(&domain.User{})

	// 3) Build and execute GORM query using that filter.
	err = gormfilter.BuildGormQuery(query, interface{}(filter))
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4) Execute
	var users []domain.User
	if err = query.Find(&users).Error; err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5) Return JSON…
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
