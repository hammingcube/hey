// Copyright (c) 2015 Elements of Programming Interviews. All rights reserved.
// Some Comments

#include <cassert>
#include <iostream>
#include <random>
#include <string>

#include "./Can_string_be_palindrome_hash.h"

using std::cout;
using std::cin;
using std::endl;
using std::string;


int main(int argc, char *argv[]) {
  string s;
  while(cin >> s){
    cout << CanStringBeAPalindromeHash::CanStringBeAPalindrome(s) << endl;
  }
  return 0;
}
