pragma solidity ^0.5.0;

// Added just for having a circular reference
import "./StorageMocks3.sol";

contract SimpleStorageMock {
  uint256 public my_public_uint256;
  string internal my_internal_string;
  uint8 private my_private_uint8;
  int8 private my_private_uint16;
  bool private my_private_bool;
  uint private my_private_uint;
  address private my_private_address;
}

contract StorageMockWithBytes {
  bytes internal my_bytes;
  bytes8 internal my_bytes8;
  bytes32 internal my_bytes32;
}

contract StorageMockWithConstants {
  uint256 public constant my_public_uint256 = 256;
  string internal constant my_internal_string = "foo";
  uint8 private constant my_private_uint8 = 8;
}

contract StorageMockWithArrays {
  uint256[] public my_public_uint256_dynarray;
  string[] internal my_internal_string_dynarray;
  address[] private my_private_address_dynarray;
  int8[10] public my_public_int8_staticarray;
  bool[20] internal my_internal_bool_staticarray;
  uint[30] private my_private_uint_staticarray;
}

contract StorageMockWithMappings {
  mapping(uint256 => string) public my_mapping;
  mapping(uint256 => mapping(string => address)) internal my_nested_mapping;
  mapping(uint256 => bool[]) private my_mapping_with_arrays;
}

contract StorageMockWithFunctions {
  function(uint) internal my_fun;
  function(string memory, string memory)[] internal my_fun_dynarray;
  function(uint) returns (address)[10] internal my_fun_staticarray;
  mapping(uint256 => function(bool)) internal my_fun_mapping;
}

contract StorageMockWithContracts {
  SimpleStorageMock public my_contract;
  SimpleStorageMock[] private my_contract_dynarray;
  SimpleStorageMock[10] internal my_contract_staticarray;
  mapping(uint256 => SimpleStorageMock) private my_contract_mapping;
  mapping(uint256 => SimpleStorageMock[]) private my_contract_dynarray_mapping;
  mapping(uint256 => SimpleStorageMock[10]) private my_contract_staticarray_mapping;
}

contract StorageMockWithStructs {
  struct MyStruct {
    uint256 struct_uint256;
    string struct_string;
    address struct_address;
  }  

  MyStruct internal my_struct;
  MyStruct[] private my_struct_dynarray;
  MyStruct[10] internal my_struct_staticarray;
  mapping(uint256 => MyStruct) private my_struct_mapping;
}

contract StorageMockWithEnums {
  enum MyEnum { State1, State2 }
 
  MyEnum public my_enum;
  MyEnum[] internal my_enum_dynarray;
  MyEnum[10] internal my_enum_staticarray;
  mapping(uint256 => MyEnum) private my_enum_mapping;
}

contract StorageMockWithComplexStructs {
  struct MyStruct {
    uint256[] uint256_dynarray;
    mapping(string => StorageMockWithEnums.MyEnum) mapping_enums;
    StorageMockWithStructs.MyStruct other_struct;
  }

  MyStruct internal my_struct;
  StorageMockWithStructs.MyStruct internal my_other_struct;
}

contract StorageMockWithRecursiveStructs {
  struct MyStruct {
    OtherStruct[] other_structs;
  }

  struct OtherStruct {
    MyStruct my_struct;
  }

  MyStruct internal my_struct;
}

contract StorageMockMixed is StorageMockWithStructs, StorageMockWithEnums, SimpleStorageMock {
}
