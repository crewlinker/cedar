use cedar_policy::*;
use once_cell::sync::OnceCell;
use std::ffi::CStr;
use std::mem;
use std::os::raw::c_char;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("failed to parse policy")]
    PolicyParsing(),
    #[error("invalid utf-8 input")]
    InvalidUtf8(),
    #[error("policies already loaded")]
    AlreadyLoaded(),
}

// allocate a block of memory and return the address to it.
#[no_mangle]
pub extern "C" fn allocate(size: usize) -> *mut c_char {
    let mut buffer = Vec::with_capacity(size);
    let ptr = buffer.as_mut_ptr();
    mem::forget(buffer);

    return ptr;
}

// #[no_mangle]
pub extern "C" fn _authorize(policy_ptr: *mut c_char, _request_ptr: *mut c_char) -> bool {
    // turn the pointer to the memory block into a cstr.
    let policy_cstr = unsafe { CStr::from_ptr(policy_ptr) };

    // libc::strcpy(policy_ptr, policy_cstr.as_ptr());
    // run the authorization logic and return true or false based on the result.
    // match maybe_authorize(policy_cstr) {
    //     Ok(result) => return result,
    //     ParseErrors(_) => return false,
    // }

    return false;

    // match policy_cstr.to_str() {
    //     Ok(policy) => {
    //         let pset = parse_policies(policy);
    //         return true;
    //     }
    //     Err(_) => {
    //         return false;
    //     }
    // }

    // policy_cstr.parse().unwrap(

    // @TODO handle policy parsing errors
    // let policy = parse_policies(policy_cstr.to_str())

    // let _request = unsafe { CStr::from_ptr(request_ptr).to_bytes().to_vec() };
}

// keep the parsed policy set in memory.
static POLICIES: OnceCell<PolicySet> = OnceCell::new();

// load the policy from a memory block. This function can only be called once
// during the lifetime of the program.
#[no_mangle]
pub extern "C" fn load_policies(policy_ptr: *mut c_char) -> u32 {
    let policy_cstr = unsafe { CStr::from_ptr(policy_ptr) };
    match _load_policies(&POLICIES, policy_cstr) {
        Ok(_) => return 0,
        Err(Error::AlreadyLoaded()) => return 1000,
        Err(Error::InvalidUtf8()) => return 1001,
        Err(Error::PolicyParsing()) => return 1002,
    }
}

// parse and set the policy from the string, this can be done only once.
fn _load_policies(cell: &OnceCell<PolicySet>, policy_cstr: &CStr) -> Result<(), Error> {
    let policy_str = str_from_cstr(policy_cstr)?;
    let loaded = parse_policies(policy_str)?;

    return cell.set(loaded).map_err(|_| Error::AlreadyLoaded());
}

// count the number of policies that have been loaded.
#[no_mangle]
pub extern "C" fn count_num_policies() -> usize {
    return _count_num_policies(&POLICIES);
}

fn _count_num_policies(cell: &OnceCell<PolicySet>) -> usize {
    return cell.get().map_or(0, |p| p.policies().count());
}

// authorize the the request against the policy and return the result.
// fn authorize(policy_cstr: &CStr) -> Result<Response, Error> {
//     let policy_str = str_from_cstr(policy_cstr)?;
//     let policies = parse_policies(policy_str)?;
//     let authorizer = Authorizer::new();

//     // formulate request
//     let action = r#"Action::"view""#.parse().unwrap();
//     let alice = r#"User::"alice""#.parse().unwrap();
//     let file = r#"File::"93""#.parse().unwrap();
//     let request = Request::new(Some(alice), Some(action), Some(file), Context::empty());

//     // formulate entities
//     let entities = Entities::empty();

//     // perform actually authorization
//     let response = authorizer.is_authorized(&request, &policies, &entities);

//     println!("{:?}", response);

//     return Ok(response);
// }

// parse the Cedar policy from the string and return the PolicySet.
fn parse_policies(policy: &str) -> Result<PolicySet, Error> {
    return policy.parse().map_err(|_| Error::PolicyParsing());
}

// convert a CStr to a &str expecting the Cstr to be valid UTF-8.
fn str_from_cstr(cstr: &CStr) -> Result<&str, Error> {
    return cstr.to_str().map_err(|_| Error::InvalidUtf8());
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_static_load_external() {
        let policys_ptr = allocate(1024);
        unsafe {
            let src = CStr::from_bytes_with_nul(b"permit(principal == User::\"alice\", action == Action::\"view\", resource == File::\"93\");\0").unwrap().as_ptr();
            std::ptr::copy_nonoverlapping(src, policys_ptr, 1024)
        };

        let code = load_policies(policys_ptr);
        assert!(code == 0, "should have loaded policies, got: {}", code);

        let count = count_num_policies();
        assert!(count == 1, "should have 1 policy loaded, got: {}", count);

        let code = load_policies(policys_ptr);
        assert!(code == 1000, "should already be loaded, got: {}", code);
    }

    #[test]
    fn test_load_policies() {
        let cell: OnceCell<PolicySet> = OnceCell::new();

        let policy_cstr = CStr::from_bytes_with_nul(b"permit(principal == User::\"alice\", action == Action::\"view\", resource == File::\"93\");\0").unwrap();
        let load1 = _load_policies(&cell, &policy_cstr);
        assert!(load1.is_ok(), "loading policies failed");
        assert!(
            _count_num_policies(&cell) == 1,
            "should have 1 policy loaded"
        );

        let load2 = _load_policies(&cell, &policy_cstr);
        assert!(load2.is_err_and(|e| e.to_string().contains("already loaded")));
    }

    #[test]
    fn test_allocate() {
        let ptr1 = allocate(1024);
        assert!(!ptr1.is_null(), "pointer is null");
    }

    #[test]
    fn test_parse_policies_error() {
        let pset1 = parse_policies("bogus");
        assert!(pset1.is_err_and(|e| e.to_string().contains("failed to parse policy")));
    }

    #[test]
    fn test_parse_policies() {
        let pset1 = parse_policies("permit(principal == User::\"alice\", action == Action::\"view\", resource == File::\"93\");");
        assert!(pset1.is_ok(), "policy parsing failed");
    }

    #[test]
    fn test_str_from_cstr() {
        let cstr = CStr::from_bytes_with_nul(b"hello\0").unwrap();
        let s = str_from_cstr(&cstr).unwrap();
        assert_eq!(s, "hello");
    }

    #[test]
    fn test_str_from_cstr_error() {
        let cstr = CStr::from_bytes_until_nul(b"Hello, \xF0\x28\x8C\xBC!\0").unwrap();
        let s = str_from_cstr(&cstr);
        assert!(s.is_err_and(|e| e.to_string().contains("invalid utf-8 input")));
    }

    // #[test]
    // fn test_parse_policies() {
    //     let pset1 = parse_policies("permit(principal == User::\"alice\", action == Action::\"view\", resource == File::\"93\");");
    //     assert!(!pset1.is_empty(), "policies are empty");
    // }
}
