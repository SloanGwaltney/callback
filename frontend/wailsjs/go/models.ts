export namespace jobs {
	
	export class NextAction {
	    text: string;
	    expectedDate: string;
	    due: string;
	
	    static createFrom(source: any = {}) {
	        return new NextAction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.expectedDate = source["expectedDate"];
	        this.due = source["due"];
	    }
	}
	export class NextInterview {
	    date: string;
	    time: string;
	    kind: string;
	    today: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NextInterview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.time = source["time"];
	        this.kind = source["kind"];
	        this.today = source["today"];
	    }
	}
	export class Card {
	    id: number;
	    company: string;
	    role: string;
	    location: string;
	    dateApplied: string;
	    interest?: number;
	    closingReason: string;
	    closingReasonText: string;
	    documentCount: number;
	    nextInterview?: NextInterview;
	    nextAction?: NextAction;
	    daysInStatus?: number;
	
	    static createFrom(source: any = {}) {
	        return new Card(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.company = source["company"];
	        this.role = source["role"];
	        this.location = source["location"];
	        this.dateApplied = source["dateApplied"];
	        this.interest = source["interest"];
	        this.closingReason = source["closingReason"];
	        this.closingReasonText = source["closingReasonText"];
	        this.documentCount = source["documentCount"];
	        this.nextInterview = this.convertValues(source["nextInterview"], NextInterview);
	        this.nextAction = this.convertValues(source["nextAction"], NextAction);
	        this.daysInStatus = source["daysInStatus"];
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
	export class Column {
	    status: string;
	    label: string;
	    count: number;
	    cards: Card[];
	
	    static createFrom(source: any = {}) {
	        return new Column(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.label = source["label"];
	        this.count = source["count"];
	        this.cards = this.convertValues(source["cards"], Card);
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
	export class Board {
	    columns: Column[];
	
	    static createFrom(source: any = {}) {
	        return new Board(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = this.convertValues(source["columns"], Column);
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
	
	
	export class Contact {
	    id: number;
	    jobId: number;
	    name: string;
	    relationship: string;
	    email: string;
	    phone: string;
	    profileLink: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new Contact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.jobId = source["jobId"];
	        this.name = source["name"];
	        this.relationship = source["relationship"];
	        this.email = source["email"];
	        this.phone = source["phone"];
	        this.profileLink = source["profileLink"];
	        this.notes = source["notes"];
	    }
	}
	export class ContactDetails {
	    name: string;
	    relationship: string;
	    email: string;
	    phone: string;
	    profileLink: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new ContactDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.relationship = source["relationship"];
	        this.email = source["email"];
	        this.phone = source["phone"];
	        this.profileLink = source["profileLink"];
	        this.notes = source["notes"];
	    }
	}
	export class Details {
	    company: string;
	    role: string;
	    description: string;
	    link: string;
	    salary: string;
	    location: string;
	    notes: string;
	    dateApplied: string;
	    interest?: number;
	
	    static createFrom(source: any = {}) {
	        return new Details(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.company = source["company"];
	        this.role = source["role"];
	        this.description = source["description"];
	        this.link = source["link"];
	        this.salary = source["salary"];
	        this.location = source["location"];
	        this.notes = source["notes"];
	        this.dateApplied = source["dateApplied"];
	        this.interest = source["interest"];
	    }
	}
	export class Document {
	    id: number;
	    jobId: number;
	    kind: string;
	    fileName: string;
	
	    static createFrom(source: any = {}) {
	        return new Document(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.jobId = source["jobId"];
	        this.kind = source["kind"];
	        this.fileName = source["fileName"];
	    }
	}
	export class Interview {
	    id: number;
	    jobId: number;
	    date: string;
	    time: string;
	    kind: string;
	    notes: string;
	    past: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Interview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.jobId = source["jobId"];
	        this.date = source["date"];
	        this.time = source["time"];
	        this.kind = source["kind"];
	        this.notes = source["notes"];
	        this.past = source["past"];
	    }
	}
	export class InterviewDetails {
	    date: string;
	    time: string;
	    kind: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new InterviewDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.time = source["time"];
	        this.kind = source["kind"];
	        this.notes = source["notes"];
	    }
	}
	export class StatusChange {
	    status: string;
	    date: string;
	    closingReason: string;
	    closingReasonText: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.date = source["date"];
	        this.closingReason = source["closingReason"];
	        this.closingReasonText = source["closingReasonText"];
	    }
	}
	export class Job {
	    id: number;
	    company: string;
	    role: string;
	    description: string;
	    link: string;
	    salary: string;
	    location: string;
	    notes: string;
	    dateApplied: string;
	    interest?: number;
	    status: string;
	    closingReason: string;
	    closingReasonOther: string;
	    closingReasonText: string;
	    nextAction?: NextAction;
	    documents: Document[];
	    contacts: Contact[];
	    interviews: Interview[];
	    statusHistory: StatusChange[];
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.company = source["company"];
	        this.role = source["role"];
	        this.description = source["description"];
	        this.link = source["link"];
	        this.salary = source["salary"];
	        this.location = source["location"];
	        this.notes = source["notes"];
	        this.dateApplied = source["dateApplied"];
	        this.interest = source["interest"];
	        this.status = source["status"];
	        this.closingReason = source["closingReason"];
	        this.closingReasonOther = source["closingReasonOther"];
	        this.closingReasonText = source["closingReasonText"];
	        this.nextAction = this.convertValues(source["nextAction"], NextAction);
	        this.documents = this.convertValues(source["documents"], Document);
	        this.contacts = this.convertValues(source["contacts"], Contact);
	        this.interviews = this.convertValues(source["interviews"], Interview);
	        this.statusHistory = this.convertValues(source["statusHistory"], StatusChange);
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
	export class LibraryDocument {
	    id: number;
	    fileName: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new LibraryDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fileName = source["fileName"];
	        this.description = source["description"];
	    }
	}
	export class Move {
	    jobId: number;
	    status: string;
	    index: number;
	    closingReason: string;
	    closingReasonOther: string;
	
	    static createFrom(source: any = {}) {
	        return new Move(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.status = source["status"];
	        this.index = source["index"];
	        this.closingReason = source["closingReason"];
	        this.closingReasonOther = source["closingReasonOther"];
	    }
	}
	
	

}

