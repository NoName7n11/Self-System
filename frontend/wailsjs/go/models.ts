export namespace domain {
	
	export class Category {
	    ID: string;
	    Name: string;
	    Description: string;
	    Source: string;
	    AcceptCount: number;
	    OverrideCount: number;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.Description = source["Description"];
	        this.Source = source["Source"];
	        this.AcceptCount = source["AcceptCount"];
	        this.OverrideCount = source["OverrideCount"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Reminder {
	    ID: string;
	    Title: string;
	    Message: string;
	    // Go type: time
	    RemindAt: any;
	    Status: string;
	    ResourceID?: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Reminder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Title = source["Title"];
	        this.Message = source["Message"];
	        this.RemindAt = this.convertValues(source["RemindAt"], null);
	        this.Status = source["Status"];
	        this.ResourceID = source["ResourceID"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResourceExtractedData {
	    extracted_title?: string;
	    extracted_description?: string;
	    main_text?: string;
	    page_type?: string;
	    classification_confidence?: number;
	    classification_source?: string;
	    needs_review?: boolean;
	    key_points?: string[];
	    entities?: string[];
	    event_date?: string;
	    deadline?: string;
	    location?: string;
	    image_type?: string;
	    image_format?: string;
	    image_width?: number;
	    image_height?: number;
	    thumbnail_base64?: string;
	    ocr_text?: string;
	    pdf_page_count?: number;
	    pdf_text?: string;
	
	    static createFrom(source: any = {}) {
	        return new ResourceExtractedData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.extracted_title = source["extracted_title"];
	        this.extracted_description = source["extracted_description"];
	        this.main_text = source["main_text"];
	        this.page_type = source["page_type"];
	        this.classification_confidence = source["classification_confidence"];
	        this.classification_source = source["classification_source"];
	        this.needs_review = source["needs_review"];
	        this.key_points = source["key_points"];
	        this.entities = source["entities"];
	        this.event_date = source["event_date"];
	        this.deadline = source["deadline"];
	        this.location = source["location"];
	        this.image_type = source["image_type"];
	        this.image_format = source["image_format"];
	        this.image_width = source["image_width"];
	        this.image_height = source["image_height"];
	        this.thumbnail_base64 = source["thumbnail_base64"];
	        this.ocr_text = source["ocr_text"];
	        this.pdf_page_count = source["pdf_page_count"];
	        this.pdf_text = source["pdf_text"];
	    }
	}
	export class Resource {
	    ID: string;
	    URL: string;
	    Host: string;
	    Title: string;
	    Summary: string;
	    CategoryID: string;
	    CategoryName: string;
	    UserOverride: boolean;
	    ExtractedData: ResourceExtractedData;
	    save_count: number;
	    archived: boolean;
	    archive_reason: string;
	    // Go type: time
	    archived_at?: any;
	    similar_to?: string[];
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Resource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.URL = source["URL"];
	        this.Host = source["Host"];
	        this.Title = source["Title"];
	        this.Summary = source["Summary"];
	        this.CategoryID = source["CategoryID"];
	        this.CategoryName = source["CategoryName"];
	        this.UserOverride = source["UserOverride"];
	        this.ExtractedData = this.convertValues(source["ExtractedData"], ResourceExtractedData);
	        this.save_count = source["save_count"];
	        this.archived = source["archived"];
	        this.archive_reason = source["archive_reason"];
	        this.archived_at = this.convertValues(source["archived_at"], null);
	        this.similar_to = source["similar_to"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Todo {
	    ID: string;
	    Title: string;
	    Details: string;
	    Status: string;
	    // Go type: time
	    DueAt?: any;
	    ResourceID?: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Todo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Title = source["Title"];
	        this.Details = source["Details"];
	        this.Status = source["Status"];
	        this.DueAt = this.convertValues(source["DueAt"], null);
	        this.ResourceID = source["ResourceID"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

