export namespace jobs {
	
	export class Card {
	    id: number;
	    company: string;
	    role: string;
	    location: string;
	    dateApplied: string;
	    closingReason: string;
	    closingReasonText: string;
	    documentCount: number;
	
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
	        this.closingReason = source["closingReason"];
	        this.closingReasonText = source["closingReasonText"];
	        this.documentCount = source["documentCount"];
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
	
	
	export class Details {
	    company: string;
	    role: string;
	    description: string;
	    link: string;
	    salary: string;
	    location: string;
	    notes: string;
	    dateApplied: string;
	
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
	    status: string;
	    closingReason: string;
	    closingReasonOther: string;
	    closingReasonText: string;
	    documents: Document[];
	
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
	        this.status = source["status"];
	        this.closingReason = source["closingReason"];
	        this.closingReasonOther = source["closingReasonOther"];
	        this.closingReasonText = source["closingReasonText"];
	        this.documents = this.convertValues(source["documents"], Document);
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

