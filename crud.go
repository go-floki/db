package db

import (
	"errors"
	"fmt"
	"github.com/astaxie/beego/orm"
	"github.com/go-floki/floki"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type (
	BasicCRUD struct {
	}

	PaginationResult struct {
		Offset int
		Total  int64
		Limit  int
	}

	FileUploader func(field string, file multipart.File, fileHeader *multipart.FileHeader) (string, string)
)

func (crud BasicCRUD) FindByField(field string, value interface{}, model interface{}) error {
	o := orm.NewOrm()

	qs := o.QueryTable(model)
	err := qs.Filter(field, value).One(model)
	if err != nil {
		return err
	}

	return err
}

func (crud BasicCRUD) FindAllByField(field string, value interface{}, model interface{}, entityList interface{}) error {
	o := orm.NewOrm()

	qs := o.QueryTable(model)
	_, err := qs.Filter(field, value).All(entityList)
	if err != nil {
		return err
	}

	return err
}

func (crud BasicCRUD) Get(id int64, model interface{}) error {
	return crud.FindByField("id", id, model)
}

func (crud BasicCRUD) Delete(model interface{}) (int64, error) {
	o := orm.NewOrm()

	TriggerEntityEvent(model, "beforeDelete", model)

	count, err := o.Delete(model)

	if err == nil {
		TriggerEntityEvent(model, "afterDelete", model)
	}

	return count, err
}

func (crud BasicCRUD) Create(model interface{}) (int64, error) {
	o := orm.NewOrm()
	id, err := o.Insert(model)
	return id, err
}

func (crud BasicCRUD) Save(model interface{}) (int64, error) {
	o := orm.NewOrm()

	TriggerEntityEvent(model, "beforeSave", model)

	count, err := o.Update(model)

	if err == nil {
		TriggerEntityEvent(model, "afterSave", model)
	}

	return count, err
}

func (crud BasicCRUD) List(model interface{}) orm.QuerySeter {
	o := orm.NewOrm()
	return o.QueryTable(model)
}

func (crud BasicCRUD) Paginate(query orm.QuerySeter, result interface{}, page int, perPage int) (int64, error) {
	offset := (page - 1) * perPage

	_ = offset
	//fmt.Println("offset:", offset, query)
	count, err := query.Offset(offset).Limit(perPage).All(result)

	return count, err
}

func (crud BasicCRUD) FetchByQuery(model interface{}, getParams url.Values, result interface{}) (PaginationResult, error) {
	query := crud.List(model)

	presult := PaginationResult{Offset: 0}

	///

	filterBy := getParams.Get("filter")
	if filterBy != "" {
		parts := strings.Split(filterBy, "__")

		key := parts[0] + "__" + parts[1]
		//op := parts[1]

		value, _ := url.QueryUnescape(parts[2])

		switch value {
		case "true":
			query = query.Filter(parts[0], true)
		case "false":
			query = query.Filter(parts[0], false)
		default:
			query = query.Filter(key, value)
		}

	}

	totalCount, err := query.Count()

	presult.Total = totalCount

	//
	offset := getParams.Get("offset")

	if offset != "" {
		offsetI, err := strconv.Atoi(offset)
		if err == nil {
			query = query.Offset(offsetI)
			presult.Offset = offsetI
		}
	}

	limit := getParams.Get("limit")
	limitI := 10
	if limit != "" {
		limitI, _ = strconv.Atoi(limit)
	}

	//
	page := getParams.Get("page")

	if page != "" {
		pageI, err := strconv.Atoi(page)
		if err == nil {
			offsetI := (pageI - 1) * limitI
			query = query.Offset(offsetI)
			presult.Offset = offsetI
		}
	}

	query = query.Limit(limitI)

	orderBy := getParams.Get("order")
	if orderBy == "" {
		orderBy = "-Id"
	}

	query = query.OrderBy(orderBy)

	presult.Limit = limitI

	query.All(result)

	return presult, err

}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func SetModelField(model reflect.Value, idx int, value string) error {
	rfield := model.Field(idx)

	switch rfield.Kind() {
	case reflect.Int:
		ivalue, err := strconv.Atoi(value)
		if err == nil {
			rfield.SetInt(int64(ivalue))
		}

	case reflect.Int64:
		ivalue, err := strconv.Atoi(value)
		if err == nil {
			rfield.SetInt(int64(ivalue))
		}

	case reflect.Int32:
		ivalue, err := strconv.Atoi(value)
		if err == nil {
			rfield.SetInt(int64(ivalue))
		}

	case reflect.Int16:
		ivalue, err := strconv.Atoi(value)
		if err == nil {
			rfield.SetInt(int64(ivalue))
		}

	case reflect.Float32:
		fvalue, err := strconv.ParseFloat(value, 32)
		if err == nil {
			rfield.SetFloat(fvalue)
		}

	case reflect.Float64:
		fvalue, err := strconv.ParseFloat(value, 64)
		if err == nil {
			rfield.SetFloat(fvalue)
		}

	case reflect.Bool:
		if value == "true" {
			rfield.SetBool(true)
		} else {
			rfield.SetBool(false)
		}

	case reflect.String:
		if value != "" {
			rfield.SetString(value)
		}

	case reflect.Struct:
		if value != "" {
			switch t := rfield.Interface().(type) {

			//
			case time.Time:
				valLen := len(value)
				inVal := value
				format := ""

				switch valLen {
				case 19:
					format = "2006-01-02T15:04:05"
					inVal = value[:min(valLen, 19)]
				case 16:
					format = "2006-01-02T15:04"
					inVal = value[:min(valLen, 16)]
				case 10:
					format = "2006-01-02"
					inVal = value[:min(valLen, 10)]
				}

				if format != "" {
					val, err := time.ParseInLocation(format, inVal, floki.TimeZone)
					if err == nil {
						rfield.Set(reflect.ValueOf(val))
						fmt.Println("parsed:", inVal, "=>", val, "using format:", format)
					} else {
						fmt.Println("error parsing date '", value, "':", err)
					}

				} else {
					fmt.Println("invalid data:", value)
				}

			default:
				fmt.Println("don't know how to parse:", t)
			}
		}

	default:
		fmt.Println("can't set field of type:", rfield.Kind())
		return errors.New("unhandled field type: " + string(rfield.Kind()))
	}

	return nil

}

func (crud BasicCRUD) ParseSubEntity(modelPtr interface{}, form url.Values, prefix string, idx int) int {
	rval := reflect.ValueOf(modelPtr).Elem()
	typ := rval.Type()
	count := 0

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)

		if p.Name == "Id" {
			continue
		}

		fieldName := prefix + "[" + strconv.Itoa(idx) + "][" + p.Name + "]"
		value := form.Get(fieldName)
		if value != "" {
			count = count + 1
		}

		err := SetModelField(rval, i, value)
		if err != nil {
			fmt.Println("can't set value for field", fieldName, ":", err.Error())
		}
	}

	return count
}

func (crud BasicCRUD) FixDatesInEntity(modelPtr interface{}) {
	rval := reflect.ValueOf(modelPtr).Elem()
	typ := rval.Type()

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)

		if p.Name == "Id" {
			continue
		}

		rfield := rval.Field(i)

		switch rfield.Kind() {
		case reflect.Struct:
			switch t := rfield.Interface().(type) {

			//
			case time.Time:
				rfield.Set(reflect.ValueOf(t.In(floki.TimeZone)))

			}

		}
	}
}

func (crud BasicCRUD) FromForm(modelPtr interface{}, form url.Values) {
	rval := reflect.ValueOf(modelPtr).Elem()
	typ := rval.Type()

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)

		if p.Name == "Id" {
			continue
		}

		SetModelField(rval, i, form.Get(p.Name))
	}
}

func (crud BasicCRUD) UploadFiles(modelPtr interface{}, req *http.Request, uploader FileUploader) {
	fmt.Println("in upload files")
	if req.MultipartForm == nil {
		fmt.Println("crud.UploadFiles(): not a multipart form!")
		return
	}

	fieldMap := make(map[string]reflect.Value)
	rval := reflect.ValueOf(modelPtr).Elem()
	typ := rval.Type()

	// create field map
	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)

		if p.Name == "Id" {
			continue
		}

		fieldMap[p.Name] = rval.Field(i)
	}

	log.Println("uploading..")

	// process files
	for key, _ := range req.MultipartForm.File {
		fileField, fileHeader, err := req.FormFile(key)
		if err != nil {
			fmt.Println(err)
			return
		}

		defer fileField.Close()

		_ = fileHeader

		rootDir, generatedName := uploader(key, fileField, fileHeader)

		// remove old file
		oldFile := fieldMap[key].String()
		oldFile = rootDir + oldFile
		if _, err := os.Stat(oldFile); err == nil {
			os.Remove(oldFile)
			fmt.Println("removed old image:", oldFile)
		}

		// persist uploaded path in model
		fieldMap[key].SetString(generatedName)

		fmt.Println("uploading to:", rootDir+generatedName)

		out, err := os.Create(rootDir + generatedName)
		if err != nil {
			fmt.Printf("Unable to create the file for writing. Check your write access privilege")
			return
		}

		defer out.Close()

		// write the content from POST to the file
		_, err = io.Copy(out, fileField)
		if err != nil {
			fmt.Println(err)
		}
	}
}
