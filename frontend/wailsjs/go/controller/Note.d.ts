// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {controller} from '../models';

export function CreateFile(arg1:string,arg2:string):Promise<void>;

export function CreateFolder(arg1:string,arg2:string):Promise<void>;

export function DeleteItem(arg1:string):Promise<void>;

export function GetFiles(arg1:string):Promise<Array<controller.FileInfo>>;

export function GetNotesDir():Promise<string>;

export function OpenDirectory():Promise<string>;

export function ReadFile(arg1:string):Promise<string>;

export function RenameItem(arg1:string,arg2:string):Promise<void>;

export function SaveFile(arg1:string,arg2:string):Promise<void>;
